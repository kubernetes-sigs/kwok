/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapshot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// LoadConfig is the a combination of the impersonation config
type LoadConfig struct {
	Clientset client.Clientset
	Filters   []*meta.RESTMapping
	NoFilers  bool
}

type uniqueKey struct {
	APIVersion string
	Kind       string
	Name       string
	UID        types.UID
}

// Loader loads the resources to cluster
// This way does not delete existing resources in the cluster,
// which will handle the ownerReference so that the resources remain relative to each other
type Loader struct {
	filterGKMap sets.Sets[schema.GroupKind]

	successCounter int
	failedCounter  int

	exist   map[uniqueKey]types.UID
	pending map[uniqueKey][]*unstructured.Unstructured

	restMapper    meta.RESTMapper
	dynamicClient dynamic.Interface

	loadConfig LoadConfig
}

// NewLoader creates a new snapshot Loader.
func NewLoader(loadConfig LoadConfig) (*Loader, error) {
	restMapper, err := loadConfig.Clientset.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}
	dynamicClient, err := loadConfig.Clientset.ToDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	l := &Loader{
		exist:         make(map[uniqueKey]types.UID),
		pending:       make(map[uniqueKey][]*unstructured.Unstructured),
		restMapper:    restMapper,
		dynamicClient: dynamicClient,
		loadConfig:    loadConfig,
	}

	return l, nil
}

// Load loads the resources to cluster
func (l *Loader) Load(ctx context.Context, decoder *yaml.Decoder) error {
	logger := log.FromContext(ctx)

	startTime := time.Now()

	for ctx.Err() == nil {
		obj, err := decoder.DecodeUnstructured()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logger.Warn("Failed to decode resource", "err", err)
			continue
		}

		if obj.GetKind() == recording.ResourcePatchType.Kind && obj.GetAPIVersion() == recording.ResourcePatchType.APIVersion {
			decoder.UndecodedUnstructured(obj)
			break
		}

		if !l.filterGK(obj.GroupVersionKind().GroupKind()) {
			logger.Warn("Skipped",
				"resource", "filtered",
				"kind", obj.GetKind(),
				"name", log.KObj(obj),
			)
			return nil
		}

		l.load(ctx, obj)
	}

	return l.finishLoad(ctx, startTime)
}

func (l *Loader) finishLoad(ctx context.Context, startTime time.Time) error {
	logger := log.FromContext(ctx)

	// Print the skipped resources
	pending := []*unstructured.Unstructured{}
	exist := map[types.UID]struct{}{}
	for _, pendingObjs := range l.pending {
		for _, pendingObj := range pendingObjs {
			uid := pendingObj.GetUID()
			if _, ok := exist[uid]; ok {
				continue
			}
			exist[uid] = struct{}{}
			pending = append(pending, pendingObj)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].GetUID() < pending[j].GetUID()
	})

	for _, pendingObj := range pending {
		missing := l.getMissingOwnerReferences(pendingObj)
		missingData := slices.Map(missing, func(or metav1.OwnerReference) ownerReference {
			return ownerReference{
				APIVersion: or.APIVersion,
				Kind:       or.Kind,
				Name:       or.Name,
			}
		})
		logger.Warn("Skipped",
			"resource", "missing owner",
			"missing", missingData,
			"kind", pendingObj.GetKind(),
			"name", log.KObj(pendingObj),
		)
	}

	if l.successCounter == 0 {
		return ErrNotHandled
	}

	if l.failedCounter != 0 {
		logger.Info("Load resources",
			"counter", l.successCounter+l.failedCounter,
			"successCounter", l.successCounter,
			"failedCounter", l.failedCounter,
			"elapsed", time.Since(startTime),
		)
	} else {
		logger.Info("Load resources",
			"counter", l.successCounter,
			"elapsed", time.Since(startTime),
		)
	}
	return nil
}

func (l *Loader) load(ctx context.Context, obj *unstructured.Unstructured) {
	// If the object has owner references, we need to wait until all the owner references are created.
	if ownerReferences := obj.GetOwnerReferences(); len(ownerReferences) != 0 {
		allExist := true
		for _, ownerReference := range ownerReferences {
			key := uniqueKeyFromOwnerReference(ownerReference)
			if _, ok := l.exist[key]; !ok {
				allExist = false
				l.pending[key] = append(l.pending[key], obj)
			}
		}
		// early return if not all owner references exist
		if !allExist {
			return
		}

		// update owner references
		l.updateOwnerReferences(obj)
	}

	// apply the object
	newObj := l.apply(ctx, obj)
	if newObj == nil {
		return
	}

	// Record the new uid
	key := uniqueKeyFromMetadata(obj)
	l.exist[key] = newObj.GetUID()

	// If there are pending objects waiting for this object, apply them.
	if pendingObjs, ok := l.pending[key]; ok {
		for _, pendingObj := range pendingObjs {
			// If the pending object has only one owner reference, or all the owner references exist, apply it.
			if len(pendingObj.GetOwnerReferences()) == 1 || l.hasAllOwnerReferences(pendingObj) {
				// update owner references
				l.updateOwnerReferences(pendingObj)

				// apply the object
				newObj = l.apply(ctx, pendingObj)
				if newObj != nil {
					key := uniqueKeyFromMetadata(pendingObj)
					l.exist[key] = newObj.GetUID()
				}
			}
		}
		// Remove the pending objects
		delete(l.pending, key)
	}
}

func (l *Loader) filterGK(gk schema.GroupKind) bool {
	if l.loadConfig.NoFilers {
		return true
	}
	if l.filterGKMap == nil {
		l.filterGKMap = sets.NewSets[schema.GroupKind]()
		for _, f := range l.loadConfig.Filters {
			l.filterGKMap.Insert(f.GroupVersionKind.GroupKind())
		}
	}
	_, ok := l.filterGKMap[gk]
	return ok
}

func (l *Loader) apply(ctx context.Context, obj *unstructured.Unstructured) *unstructured.Unstructured {
	gvr := obj.GroupVersionKind().GroupVersion().WithResource(obj.GetKind())

	logger := log.FromContext(ctx)
	logger = logger.With(
		"kind", obj.GetKind(),
		"name", log.KObj(obj),
	)

	err := retry.OnError(defaultRetry, discovery.IsGroupDiscoveryFailedError, func() error {
		g, err := l.restMapper.ResourceFor(gvr)
		if err != nil {
			logger.Warn("failed to get resource", "err", err)
			return err
		}
		gvr = g
		return nil
	})
	if err != nil {
		l.failedCounter++
		return nil
	}

	obj.SetResourceVersion("")

	nri := l.dynamicClient.Resource(gvr)
	var ri dynamic.ResourceInterface = nri

	if ns := obj.GetNamespace(); ns != "" {
		ri = nri.Namespace(ns)
	}

	var newObj *unstructured.Unstructured
	err = retry.OnError(defaultRetry, isNotFound, func() error {
		newObj, err = ri.Create(ctx, obj, metav1.CreateOptions{FieldValidation: "Ignore"})
		return err
	})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			l.failedCounter++
			logger.Error("Failed to create resource", err)
			return nil
		}
		newObj, err = ri.Update(ctx, obj, metav1.UpdateOptions{FieldValidation: "Ignore"})
		if err != nil {
			l.failedCounter++
			if apierrors.IsConflict(err) {
				logger.Warn("Conflict")
				return nil
			}
			logger.Error("Failed to update resource", err)
			return nil
		}
		logger.Debug("Updated")
	} else {
		logger.Debug("Created")
	}

	if newObj != nil {
		status, ok, _ := unstructured.NestedFieldNoCopy(obj.Object, "status")
		if ok && !reflect.DeepEqual(newObj.Object["status"], status) {
			newObj.Object["status"] = status
			newObj, err = ri.UpdateStatus(ctx, newObj, metav1.UpdateOptions{FieldValidation: "Ignore"})
			if err != nil {
				logger.Error("Failed to update status", err)
			}
		}
	}

	l.successCounter++
	return newObj
}

func isNotFound(err error) bool {
	return apierrors.IsNotFound(err) ||
		(apierrors.IsForbidden(err) && strings.Contains(err.Error(), "not found"))
}

var defaultRetry = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   1.0,
	Jitter:   0.1,
}

func (l *Loader) hasAllOwnerReferences(obj *unstructured.Unstructured) bool {
	ownerReferences := obj.GetOwnerReferences()
	if len(ownerReferences) == 0 {
		return true
	}
	for _, ownerReference := range ownerReferences {
		key := uniqueKeyFromOwnerReference(ownerReference)
		if _, ok := l.exist[key]; !ok {
			return false
		}
	}
	return true
}

type ownerReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

func (l *Loader) getMissingOwnerReferences(obj *unstructured.Unstructured) []metav1.OwnerReference {
	ownerReferences := obj.GetOwnerReferences()
	if len(ownerReferences) == 0 {
		return nil
	}
	var missingOwnerReferences []metav1.OwnerReference
	for _, ownerReference := range ownerReferences {
		key := uniqueKeyFromOwnerReference(ownerReference)
		if _, ok := l.exist[key]; !ok {
			missingOwnerReferences = append(missingOwnerReferences, ownerReference)
		}
	}
	return missingOwnerReferences
}

func (l *Loader) updateOwnerReferences(obj *unstructured.Unstructured) {
	ownerReferences := obj.GetOwnerReferences()
	if len(ownerReferences) == 0 {
		return
	}
	for i := range ownerReferences {
		key := uniqueKeyFromOwnerReference(ownerReferences[i])
		ownerReference := &ownerReferences[i]
		if uid, ok := l.exist[key]; ok {
			ownerReference.UID = uid
		}
	}
	obj.SetOwnerReferences(ownerReferences)
}

func uniqueKeyFromOwnerReference(ownerReference metav1.OwnerReference) uniqueKey {
	return uniqueKey{
		APIVersion: ownerReference.APIVersion,
		Kind:       ownerReference.Kind,
		Name:       ownerReference.Name,
		UID:        ownerReference.UID,
	}
}

func uniqueKeyFromMetadata(obj *unstructured.Unstructured) uniqueKey {
	return uniqueKey{
		APIVersion: obj.GetAPIVersion(),
		Kind:       obj.GetKind(),
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
	}
}
