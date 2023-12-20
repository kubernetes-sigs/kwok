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
	"sync/atomic"
	"time"

	"github.com/wzshiming/getch"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// LoadConfig is the a combination of the impersonation config
type LoadConfig struct {
	Filters  []*meta.RESTMapping
	NoFilers bool
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
	filterGRMap sets.Sets[schema.GroupResource]

	successCounter int
	failedCounter  int

	exist   map[uniqueKey]types.UID
	pending map[uniqueKey][]*unstructured.Unstructured

	restMapper    meta.RESTMapper
	dynamicClient dynamic.Interface

	loadConfig LoadConfig

	pause         atomic.Bool
	isTerminal    bool
	replaySpeed   atomic.Pointer[speed]
	delayingQueue queue.DelayingQueue[patchMeta]
}

// NewLoader creates a new snapshot Loader.
func NewLoader(clientset client.Clientset, loadConfig LoadConfig) (*Loader, error) {
	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}
	dynamicClient, err := clientset.ToDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	l := &Loader{
		exist:         make(map[uniqueKey]types.UID),
		pending:       make(map[uniqueKey][]*unstructured.Unstructured),
		restMapper:    restMapper,
		dynamicClient: dynamicClient,
		loadConfig:    loadConfig,
		isTerminal:    log.IsTerminal(),
	}
	l.replaySpeed.Store(format.Ptr(speed(1)))

	return l, nil
}

func (l *Loader) Load(ctx context.Context, decoder *yaml.Decoder) error {
	logger := log.FromContext(ctx)

	startTime := time.Now()

	for ctx.Err() == nil {
		obj, err := decoder.DecodeUnstructured()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to decode object: %w", err)
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

func (l *Loader) Replay(ctx context.Context, decoder *yaml.Decoder) error {
	logger := log.FromContext(ctx)

	if l.isTerminal {
		logger.Info("Press `Space` key to pause, press `Enter` key to continue")
		logger.Info("Press `U` key to speed up, press `D` key to speed down")
		go l.handleInput(ctx)
	}

	var dur time.Duration
	for ctx.Err() == nil {
		obj, err := decoder.DecodeUnstructured()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to decode object: %w", err)
		}
		if obj.GetKind() != recording.ResourcePatchType.Kind || obj.GetAPIVersion() != recording.ResourcePatchType.APIVersion {
			continue
		}

		resourcePatch, err := yaml.Convert[recording.ResourcePatch](obj)
		if err != nil {
			return err
		}

		gr := schema.GroupResource{
			Group:    resourcePatch.Target.Type.Group,
			Resource: resourcePatch.Target.Type.Resource,
		}

		if !l.filterGR(gr) {
			continue
		}

		d := resourcePatch.DurationNanosecond - dur

		if d <= 0 {
			l.addPatch(ctx, obj, &resourcePatch, 0)
			continue
		}

		l.addPatch(ctx, obj, &resourcePatch, d)
		dur = resourcePatch.DurationNanosecond
	}

	for l.delayingQueue.Len() != 0 || l.delayingQueue.Pending() != 0 {
		time.Sleep(time.Second)
	}

	return nil
}

func (l *Loader) handlePause(ctx context.Context) {
	if !l.isTerminal {
		return
	}
	for l.pause.Load() {
		if err := ctx.Err(); err != nil {
			return
		}
		time.Sleep(time.Second / 10)
	}
}

func (l *Loader) handleInput(ctx context.Context) {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		r, _, err := getch.Getch()
		if err != nil {
			logger.Error("Failed to get key", err)
			return
		}
		switch r {
		case getch.Key_u, getch.KeyU:
			s := *l.replaySpeed.Load()
			s = s.Up()
			l.replaySpeed.Store(&s)
			logger.Info("Speed up", "rate", s)
		case getch.Key_d, getch.KeyD:
			s := *l.replaySpeed.Load()
			s = s.Down()
			if s != 0 {
				l.replaySpeed.Store(&s)
			}
			logger.Info("Speed down", "rate", s)
		case getch.KeySpace:
			if !l.pause.Load() {
				l.pause.Store(true)
				logger.Info("Paused, Press `Enter` key to continue")
			} else {
				logger.Info("Already paused, Press `Enter` key to continue")
			}
		case getch.KeyCtrlJ:
			if l.pause.Load() {
				logger.Info("Continue, Press `Space` key to pause")
				l.pause.Store(false)
			} else {
				logger.Info("Already running, Press `Space` key to pause")
			}
		default:
			logger.Warn("Unknown key", "key", r)
		}
	}
}

func (l *Loader) addPatch(ctx context.Context, obj *unstructured.Unstructured, rpatch *recording.ResourcePatch, duration time.Duration) {
	if l.delayingQueue == nil {
		l.delayingQueue = queue.NewDelayingQueue[patchMeta](clock.RealClock{})
		for i := 0; i != 10; i++ {
			go l.patchWorker(ctx, l.delayingQueue)
		}
	}

	block := 0
	logger := log.FromContext(ctx)
	for ; l.delayingQueue.Len() > 1000; block++ {
		if err := ctx.Err(); err != nil {
			return
		}

		time.Sleep(time.Second / 10)
	}
	l.handlePause(ctx)

	if block != 0 {
		s := *l.replaySpeed.Load()
		s = s.Down()
		if s != 0 {
			l.replaySpeed.Store(&s)
		}
		logger.Info("Speed down because of blocking", "rate", s)
	}

	if s := float64(*l.replaySpeed.Load()); s != 1 && s > 0 {
		duration = time.Duration(float64(duration) / s)
	}

	for duration > time.Second {
		if err := ctx.Err(); err != nil {
			return
		}

		time.Sleep(time.Second / 10)
		duration -= time.Second / 10
	}

	p := patchMeta{resourcePatch: rpatch, obj: obj}
	if duration <= 0 {
		l.delayingQueue.Add(p)
	} else {
		l.delayingQueue.AddAfter(p, duration)
	}
}

func (l *Loader) patchWorker(ctx context.Context, q queue.DelayingQueue[patchMeta]) {
	for {
		p, ok := q.GetOrWaitWithContext(ctx)
		if !ok {
			return
		}
		l.patch(ctx, p)
	}
}

type patchMeta struct {
	resourcePatch *recording.ResourcePatch
	obj           *unstructured.Unstructured
}

func (l *Loader) patch(ctx context.Context, patchMeta patchMeta) {
	logger := log.FromContext(ctx)
	resourcePatch := patchMeta.resourcePatch
	obj := patchMeta.obj
	gvr := resourcePatch.GetTargetGroupVersionResource()

	name, ns := resourcePatch.GetTargetName()
	nri := l.dynamicClient.Resource(gvr)
	var ri dynamic.ResourceInterface = nri

	if ns != "" {
		ri = nri.Namespace(ns)
	}

	switch {
	case resourcePatch.Delete:
		err := ri.Delete(ctx, name, metav1.DeleteOptions{GracePeriodSeconds: format.Ptr[int64](0)})
		if err != nil {
			logger.Warn("Failed to delete resource",
				"err", err,
				"gvr", gvr,
				"name", log.KObj(obj),
				"target", resourcePatch.Target,
			)
		}
	case resourcePatch.Create:
		obj := &unstructured.Unstructured{}
		err := obj.UnmarshalJSON(resourcePatch.Patch)
		if err != nil {
			logger.Error("Failed to unmarshal resource", err,
				"gvr", gvr,
				"name", log.KObj(obj),
			)
			return
		}

		l.updateOwnerReferences(obj)
		obj.SetUID("")
		newObj, err := ri.Create(ctx, obj, metav1.CreateOptions{
			FieldValidation: "Ignore",
		})
		if err != nil {
			logger.Warn("Failed to create resource",
				"err", err,
				"gvr", gvr,
				"name", log.KObj(obj),
				"target", resourcePatch.Target,
				"obj", obj,
			)
		}
		if newObj != nil {
			status, ok, _ := unstructured.NestedFieldNoCopy(obj.Object, "status")
			if ok && !reflect.DeepEqual(newObj.Object["status"], status) {
				newObj.Object["status"] = status
				_, err = ri.UpdateStatus(ctx, newObj, metav1.UpdateOptions{FieldValidation: "Ignore"})
				if err != nil {
					logger.Warn("Failed to update status", "err", err)
				}
			}
		}
	case len(resourcePatch.Patch) != 0:
		if gvr.Resource == "pods" && gvr.Version == "v1" {
			nodeName, ok, err := unstructured.NestedString(obj.Object, "patch", "spec", "nodeName")
			// For scheduling, we need to use a binding for the pod.
			if err == nil && ok && nodeName != "" {
				bind := &unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "v1",
						"kind":       "Binding",
						"metadata": map[string]any{
							"name":      name,
							"namespace": ns,
						},
						"target": map[string]any{
							"apiVersion": "v1",
							"kind":       "Node",
							"name":       nodeName,
						},
					},
				}
				_, err = ri.Create(ctx, bind, metav1.CreateOptions{
					FieldValidation: "Ignore",
				}, "binding")
				if err != nil {
					logger.Warn("Failed to create binding",
						"err", err,
						"gvr", gvr,
						"name", log.KObj(obj),
						"target", resourcePatch.Target,
					)
				}
				return
			}
		}

		var subresource string
		_, hasStatus, _ := unstructured.NestedFieldNoCopy(obj.Object, "patch", "status")
		if hasStatus {
			subresource = "status"
		}
		_, err := ri.Patch(ctx, name, types.StrategicMergePatchType, resourcePatch.Patch, metav1.PatchOptions{
			FieldValidation: "Ignore",
		}, subresource)
		if err != nil {
			logger.Warn("Failed to patch resource",
				"err", err,
				"gvr", gvr,
				"name", log.KObj(obj),
				"target", resourcePatch.Target,
			)
		}
	}
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

func (l *Loader) filterGR(gr schema.GroupResource) bool {
	if l.loadConfig.NoFilers {
		return true
	}
	if l.filterGRMap == nil {
		l.filterGRMap = sets.NewSets[schema.GroupResource]()
		for _, f := range l.loadConfig.Filters {
			l.filterGRMap.Insert(f.Resource.GroupResource())
		}
	}
	_, ok := l.filterGRMap[gr]
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
