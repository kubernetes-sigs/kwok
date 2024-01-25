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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/log"
	clientset "sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/heap"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// SaveConfig is the combination of the impersonation config
// and the PagerConfig.
type SaveConfig struct {
	Clientset clientset.Clientset
	Client    Client
	Prefix    string
}

// Saver is a snapshot saver.
type Saver struct {
	restMapper      meta.RESTMapper
	patchMetaSchema *patch.PatchMetaFromOpenAPI3

	rev        int64
	saveConfig SaveConfig
	track      map[log.ObjectRef]json.RawMessage
	baseTime   time.Time
	clock      clock.PassiveClock
}

// NewSaver creates a new snapshot saver.
func NewSaver(saveConfig SaveConfig) (*Saver, error) {
	restMapper, err := saveConfig.Clientset.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}

	restConfig, err := saveConfig.Clientset.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	restConfig.GroupVersion = &schema.GroupVersion{}
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client: %w", err)
	}

	patchMetaSchema := patch.NewPatchMetaFromOpenAPI3(restClient)

	return &Saver{
		saveConfig:      saveConfig,
		restMapper:      restMapper,
		patchMetaSchema: patchMetaSchema,
		track:           map[log.ObjectRef]json.RawMessage{},
		clock:           clock.RealClock{},
	}, nil
}

func (s *Saver) save(encoder *yaml.Encoder, kv *KeyValue) error {
	value := kv.Value
	if value == nil {
		value = kv.PrevValue
	}
	inMediaType, err := DetectMediaType(value)
	if err != nil {
		return err
	}
	_, data, err := Convert(inMediaType, JSONMediaType, value)
	if err != nil {
		return err
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	if obj.GetName() == "" {
		return nil
	}

	err = encoder.Encode(obj)
	if err != nil {
		return err
	}

	s.track[log.KObj(obj)] = data
	return nil
}

// Save saves the snapshot of cluster
func (s *Saver) Save(ctx context.Context, encoder *yaml.Encoder) error {
	rev, err := s.saveConfig.Client.Get(ctx, s.saveConfig.Prefix,
		WithResponse(func(kv *KeyValue) error {
			return s.save(encoder, kv)
		}),
	)
	if err != nil {
		return err
	}

	s.rev = rev
	return nil
}

func (s *Saver) buildResourcePatch(kv *KeyValue) (*recording.ResourcePatch, error) {
	lastValue := kv.Value
	if lastValue == nil {
		lastValue = kv.PrevValue
	}
	inMediaType, err := DetectMediaType(lastValue)
	if err != nil {
		return nil, err
	}
	_, data, err := Convert(inMediaType, JSONMediaType, lastValue)
	if err != nil {
		return nil, err
	}
	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}

	if obj.GetName() == "" {
		return nil, nil
	}

	gvk := obj.GroupVersionKind()
	gk := gvk.GroupKind()

	restMapping, err := s.restMapper.RESTMapping(gk)
	if err != nil {
		return nil, err
	}
	gvr := restMapping.Resource
	gvr.Version = gvk.Version

	rp := recording.ResourcePatch{}
	rp.TypeMeta = recording.ResourcePatchType
	rp.SetTargetGroupVersionResource(gvr)
	rp.SetTargetName(obj.GetName(), obj.GetNamespace())

	// base time is the time of the first object
	if s.baseTime.IsZero() {
		now := s.clock.Now()
		s.baseTime = now
	} else {
		now := s.clock.Now()
		rp.SetDuration(now.Sub(s.baseTime))
	}

	switch {
	case kv.Value != nil:
		patchMeta, err := s.patchMetaSchema.Lookup(gvr)
		if err != nil {
			s, err0 := PatchMetaFromStruct(restMapping.GroupVersionKind)
			if err0 != nil {
				return nil, err
			}
			patchMeta = s
		}

		err = rp.SetContent(obj, s.track, patchMeta)
		if err != nil {
			return nil, err
		}
	default:
		rp.SetDelete(obj, s.track)
	}

	return &rp, nil
}

// Record records the snapshot of cluster.
func (s *Saver) Record(ctx context.Context, encoder *yaml.Encoder) error {
	h := heap.NewHeap[time.Duration, *recording.ResourcePatch]()

	err := s.saveConfig.Client.Watch(ctx, s.saveConfig.Prefix,
		WithRevision(s.rev),
		WithResponse(func(kv *KeyValue) error {
			rp, err := s.buildResourcePatch(kv)
			if err != nil {
				return err
			}
			if rp == nil {
				return nil
			}

			h.Push(rp.DurationNanosecond, rp)

			// Tolerate events that are out of order over a period of time
			if h.Len() >= 128 {
				_, rp, _ := h.Pop()
				err = encoder.Encode(rp)
				if err != nil {
					return err
				}
			}
			return nil
		}),
	)
	if err != nil {
		return err
	}

	// Flush the remaining events
	for {
		_, rp, ok := h.Pop()
		if !ok {
			break
		}
		err = encoder.Encode(rp)
		if err != nil {
			return err
		}
	}
	return nil
}
