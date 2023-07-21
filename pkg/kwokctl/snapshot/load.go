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
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// Load loads the resources to cluster from the reader
func Load(ctx context.Context, clientset client.Clientset, r io.Reader, filters []string) error {
	l, err := newLoader(clientset, filters == nil)
	if err != nil {
		return err
	}
	l.addResource(ctx, filters)

	return l.Load(ctx, r)
}

type uniqueKey struct {
	APIVersion string
	Kind       string
	Name       string
	UID        types.UID
}

// loader loads the resources to cluster
// This way does not delete existing resources in the cluster,
// which will handle the ownerReference so that the resources remain relative to each other
type loader struct {
	filterMap map[schema.GroupKind]struct{}

	successCounter int
	failedCounter  int

	exist   map[uniqueKey]types.UID
	pending map[uniqueKey][]*unstructured.Unstructured

	restMapper    meta.RESTMapper
	dynamicClient dynamic.Interface
}

func newLoader(clientset client.Clientset, noFilter bool) (*loader, error) {
	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}
	dynClient, err := clientset.ToDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	l := &loader{
		exist:         make(map[uniqueKey]types.UID),
		pending:       make(map[uniqueKey][]*unstructured.Unstructured),
		restMapper:    restMapper,
		dynamicClient: dynClient,
	}
	if !noFilter {
		l.filterMap = make(map[schema.GroupKind]struct{})
	}
	return l, nil
}

func (l *loader) addResource(ctx context.Context, resources []string) {
	if l.filterMap == nil {
		return
	}
	logger := log.FromContext(ctx)
	for _, resource := range resources {
		mapping, err := mappingFor(l.restMapper, resource)
		if err != nil {
			logger.Warn("Failed to get mapping for resource", "resource", resource, "err", err)
			continue
		}
		l.filterMap[mapping.GroupVersionKind.GroupKind()] = struct{}{}
	}
}

func (l *loader) Load(ctx context.Context, r io.Reader) error {
	logger := log.FromContext(ctx)

	start := time.Now()
	decoder := yaml.NewDecoder(r)

	err := decoder.Decode(func(obj *unstructured.Unstructured) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !l.filter(obj) {
			logger.Warn("Skipped",
				"resource", "filtered",
				"kind", obj.GetKind(),
				"name", log.KObj(obj),
			)
			return nil
		}

		l.load(ctx, obj)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to decode objects: %w", err)
	}

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
			"elapsed", time.Since(start),
		)
	} else {
		logger.Info("Load resources",
			"counter", l.successCounter,
			"elapsed", time.Since(start),
		)
	}
	return nil
}

func (l *loader) load(ctx context.Context, obj *unstructured.Unstructured) {
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

func (l *loader) filter(obj *unstructured.Unstructured) bool {
	if l.filterMap == nil {
		return true
	}
	_, ok := l.filterMap[obj.GroupVersionKind().GroupKind()]
	return ok
}

func (l *loader) apply(ctx context.Context, obj *unstructured.Unstructured) *unstructured.Unstructured {
	gvr := obj.GroupVersionKind().GroupVersion().WithResource(obj.GetKind())

	logger := log.FromContext(ctx)
	logger = logger.With(
		"kind", obj.GetKind(),
		"name", log.KObj(obj),
	)

	gvr, err := l.restMapper.ResourceFor(gvr)
	if err != nil {
		l.failedCounter++
		logger.Error("Failed to get resource", err)
		return nil
	}

	clearUnstructured(obj)

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

func (l *loader) hasAllOwnerReferences(obj *unstructured.Unstructured) bool {
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

func (l *loader) getMissingOwnerReferences(obj *unstructured.Unstructured) []metav1.OwnerReference {
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

func (l *loader) updateOwnerReferences(obj *unstructured.Unstructured) {
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
