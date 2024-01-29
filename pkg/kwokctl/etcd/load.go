/*
Copyright 2024 The Kubernetes Authors.

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

package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/action/v1alpha1"
	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/log"
	clientset "sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/heap"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// LoadConfig is the combination of the impersonation config
type LoadConfig struct {
	Clientset clientset.Clientset
	Client    Client
	Prefix    string
}

// Loader loads the resources to cluster
// This way does not delete existing resources in the cluster,
// which will Handle the ownerReference so that the resources remain relative to each other
type Loader struct {
	tracksData map[schema.GroupVersionResource]map[log.ObjectRef]json.RawMessage

	loadConfig LoadConfig

	restMapper      meta.RESTMapper
	patchMetaSchema *patch.PatchMetaFromOpenAPI3

	handle *recording.Handle
	clock  clock.Clock
}

// NewLoader creates a new snapshot Loader.
func NewLoader(loadConfig LoadConfig) (*Loader, error) {
	restMapper, err := loadConfig.Clientset.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}

	restConfig, err := loadConfig.Clientset.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	restConfig.GroupVersion = &schema.GroupVersion{}
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client: %w", err)
	}

	patchMetaSchema := patch.NewPatchMetaFromOpenAPI3(restClient)

	l := &Loader{
		tracksData:      map[schema.GroupVersionResource]map[log.ObjectRef]json.RawMessage{},
		restMapper:      restMapper,
		loadConfig:      loadConfig,
		patchMetaSchema: patchMetaSchema,
		clock:           clock.RealClock{},
	}

	return l, nil
}

// AllowHandle allows the handle to be used
func (l *Loader) AllowHandle(ctx context.Context) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	handle := recording.NewHandle()

	handle.Info(ctx)
	go handle.Input(ctx)

	l.handle = handle

	return func() {
		l.handle = nil
		cancel()
	}
}

// Load loads the resources to cluster
func (l *Loader) Load(ctx context.Context, decoder *yaml.Decoder) error {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		obj, err := decoder.DecodeUnstructured()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logger.Warn("failed to decode object",
				"err", err,
			)
			continue
		}

		if obj.GetKind() == recording.ResourcePatchType.Kind && obj.GetAPIVersion() == recording.ResourcePatchType.APIVersion {
			// Leave the patch to the replay function
			decoder.UndecodedUnstructured(obj)
			break
		}

		err = l.applyResource(ctx, obj)
		if err != nil {
			logger.Warn("failed to apply resource",
				"err", err,
				"kind", obj.GetKind(),
				"name", log.KObj(obj),
			)
		}
	}

	return nil
}

// Replay replays the resources to cluster
func (l *Loader) Replay(ctx context.Context, decoder *yaml.Decoder) error {
	logger := log.FromContext(ctx)

	h := heap.NewHeap[time.Duration, *recording.ResourcePatch]()

	var dur time.Duration
	for ctx.Err() == nil {
		obj, err := decoder.DecodeUnstructured()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logger.Warn("failed to decode object",
				"err", err,
			)
			continue
		}
		if obj.GetKind() != recording.ResourcePatchType.Kind || obj.GetAPIVersion() != recording.ResourcePatchType.APIVersion {
			logger.Warn("unexpected object",
				"kind", obj.GetKind(),
				"apiVersion", obj.GetAPIVersion(),
			)
			continue
		}

		resourcePatch, err := yaml.Convert[recording.ResourcePatch](obj)
		if err != nil {
			return err
		}

		h.Push(resourcePatch.DurationNanosecond, &resourcePatch)

		// Tolerate events that are out of order over a period of time
		if h.Len() >= 1024 {
			_, rp, _ := h.Pop()
			l.handleResourcePatch(ctx, rp, &dur)
		}
	}

	// Flush the remaining events
	for ctx.Err() == nil {
		_, rp, ok := h.Pop()
		if !ok {
			break
		}
		l.handleResourcePatch(ctx, rp, &dur)
	}

	return nil
}

func (l *Loader) handleResourcePatch(ctx context.Context, resourcePatch *recording.ResourcePatch, dur *time.Duration) {
	d := resourcePatch.DurationNanosecond - *dur
	switch {
	case d > 0:
		*dur = resourcePatch.DurationNanosecond
	case d < -1*time.Second:
		if l.handle != nil {
			logger := log.FromContext(ctx)
			sd := l.handle.SpeedDown()

			logger.Warn("Speed is too fast, speed down",
				"rate", sd,
				"over", -d,
				"current", resourcePatch.DurationNanosecond,
			)

			*dur = resourcePatch.DurationNanosecond
		}
		d = 0
	default:
		d = 0
	}

	for d > 0 {
		// Handle pause
		if l.handle != nil {
			l.handlePause(ctx)
		}

		step := time.Second
		if step > d {
			step = d
		}
		d -= step

		// Adjusting speed
		if step > 0 && l.handle != nil {
			step = time.Duration(float64(step) / float64(l.handle.Speed()))
		}
		if step > 0 {
			l.clock.Sleep(step)
		}
	}
	// Handle pause
	if l.handle != nil {
		l.handlePause(ctx)
	}

	start := l.clock.Now()
	l.applyResourcePatch(ctx, resourcePatch)
	past := l.clock.Since(start)
	if past > 0 {
		if l.handle != nil {
			past = time.Duration(float64(past) * float64(l.handle.Speed()))
		}
		*dur += past
	}
}

func (l *Loader) handlePause(ctx context.Context) {
	for l.handle.IsPause() {
		if err := ctx.Err(); err != nil {
			return
		}
		l.clock.Sleep(time.Second / 10)
	}
}

func (l *Loader) getData(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) ([]byte, error) {
	var dataStore []byte
	_, err := l.loadConfig.Client.Get(ctx, l.loadConfig.Prefix,
		WithName(name, namespace),
		WithGVR(gvr),
		WithResponse(func(kv *KeyValue) error {
			dataStore = kv.Value
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	if len(dataStore) == 0 {
		return nil, fmt.Errorf("resource not found")
	}

	mediaType, err := MediaTypeFromGVR(gvr)
	if err != nil {
		return nil, fmt.Errorf("failed to get media type: %w", err)
	}

	var data = dataStore
	if JSONMediaType != mediaType {
		_, data, err = Convert(mediaType, JSONMediaType, dataStore)
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource: %w", err)
		}
	}

	return data, nil
}

func (l *Loader) delData(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) error {
	err := l.loadConfig.Client.Delete(ctx, l.loadConfig.Prefix,
		WithName(name, namespace),
		WithGVR(gvr),
	)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	key := log.KRef(namespace, name)
	delete(l.tracksData[gvr], key)
	return nil
}

func (l *Loader) putData(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) error {
	mediaType, err := MediaTypeFromGVR(gvr)
	if err != nil {
		return fmt.Errorf("failed to get media type: %w", err)
	}

	obj.SetResourceVersion("")

	data, err := obj.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}
	dataStore := data
	if JSONMediaType != mediaType {
		_, dataStore, err = Convert(JSONMediaType, mediaType, data)
		if err != nil {
			return fmt.Errorf("failed to convert resource: %w", err)
		}
	}
	key := log.KObj(obj)
	err = l.loadConfig.Client.Put(ctx, l.loadConfig.Prefix, dataStore,
		WithName(key.Name, key.Namespace),
		WithGVR(gvr),
	)
	if err != nil {
		return fmt.Errorf("failed to put resource: %w", err)
	}

	l.tracksData[gvr][key] = data
	return nil
}

func (l *Loader) patchData(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, patchData []byte) error {
	mediaType, err := MediaTypeFromGVR(gvr)
	if err != nil {
		return fmt.Errorf("failed to get media type: %w", err)
	}

	statusPatchMeta, err := l.patchMetaSchema.Lookup(gvr)
	if err != nil {
		gvk, err0 := l.restMapper.KindFor(gvr)
		if err0 != nil {
			return fmt.Errorf("failed to lookup gvk: %w", err)
		}

		s, err0 := PatchMetaFromStruct(gvk)
		if err0 != nil {
			return fmt.Errorf("failed to lookup patch meta: %w", err)
		}
		statusPatchMeta = s
	}

	m := map[string]interface{}{}
	err = json.Unmarshal(patchData, &m)
	if err != nil {
		return fmt.Errorf("failed to unmarshal patch data: %w", err)
	}

	dataMap, err := strategicpatch.StrategicMergeMapPatchUsingLookupPatchMeta(obj.Object, m, statusPatchMeta)
	if err != nil {
		return fmt.Errorf("failed to merge patch: %w", err)
	}

	data, err := json.Marshal(dataMap)
	if err != nil {
		return fmt.Errorf("failed to marshal patch data: %w", err)
	}

	dataStore := data
	if JSONMediaType != mediaType {
		_, dataStore, err = Convert(JSONMediaType, mediaType, data)
		if err != nil {
			return fmt.Errorf("failed to convert resource: %w", err)
		}
	}

	key := log.KObj(obj)
	err = l.loadConfig.Client.Put(ctx, l.loadConfig.Prefix, dataStore,
		WithName(key.Name, key.Namespace),
		WithGVR(gvr),
	)
	if err != nil {
		return fmt.Errorf("failed to put resource: %w", err)
	}

	l.tracksData[gvr][key] = data
	return nil
}

func (l *Loader) applyResource(ctx context.Context, obj *unstructured.Unstructured) error {
	gvk := obj.GroupVersionKind()
	gk := gvk.GroupKind()

	restMapping, err := l.restMapper.RESTMapping(gk)
	if err != nil {
		return fmt.Errorf("failed to get mapping: %w", err)
	}

	gvr := restMapping.Resource
	gvr.Version = gvk.Version

	if l.tracksData[gvr] == nil {
		l.tracksData[gvr] = map[log.ObjectRef]json.RawMessage{}
	}

	err = l.putData(ctx, gvr, obj)
	if err != nil {
		return fmt.Errorf("failed to put data: %w", err)
	}
	return nil
}

func (l *Loader) applyResourcePatch(ctx context.Context, resourcePatch *recording.ResourcePatch) {
	logger := log.FromContext(ctx)

	gvr := resourcePatch.GetTargetGroupVersionResource()

	name, namespace := resourcePatch.GetTargetName()

	if l.tracksData[gvr] == nil {
		l.tracksData[gvr] = map[log.ObjectRef]json.RawMessage{}
	}

	key := log.KRef(namespace, name)
	switch resourcePatch.Method {
	case v1alpha1.PatchMethodDelete:
		err := l.delData(ctx, gvr, name, namespace)
		if err != nil {
			logger.Warn("Failed to delete resource", "err", err)
			return
		}

	case v1alpha1.PatchMethodCreate:
		obj := &unstructured.Unstructured{}
		err := obj.UnmarshalJSON(resourcePatch.Template)
		if err != nil {
			logger.Warn("Failed to unmarshal resource", "err", err)
			return
		}

		err = l.putData(ctx, gvr, obj)
		if err != nil {
			logger.Warn("Failed to put resource",
				"err", err,
				"kind", obj.GetKind(),
				"name", log.KObj(obj),
			)
			return
		}

	case v1alpha1.PatchMethodPatch:
		original := l.tracksData[gvr][key]
		if original == nil {
			origin, err := l.getData(ctx, gvr, name, namespace)
			if err != nil {
				logger.Warn("Failed to get original resource",
					"err", err,
					"gvr", gvr,
					"name", key,
				)
				return
			}
			logger.Warn("Modify a resource that is not currently created",
				"gvr", gvr,
				"name", key,
			)
			original = origin
			l.tracksData[gvr][key] = origin
		}

		obj := &unstructured.Unstructured{}
		err := obj.UnmarshalJSON(original)
		if err != nil {
			logger.Warn("Failed to unmarshal resource", "err", err)
			return
		}

		err = l.patchData(ctx, gvr, obj, resourcePatch.Template)
		if err != nil {
			logger.Warn("Failed to patch resource",
				"err", err,
				"kind", obj.GetKind(),
				"name", log.KObj(obj),
			)
			return
		}
	}
}
