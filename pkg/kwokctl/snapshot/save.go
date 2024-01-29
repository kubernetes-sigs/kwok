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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/pager"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/heap"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// PagerConfig is the configuration of the list pager.
// It defines the page size and the page buffer size of the list pager.
type PagerConfig struct {
	PageSize       int64
	PageBufferSize int32
}

// SaveConfig is the a combination of the impersonation config
// and the PagerConfig.
type SaveConfig struct {
	Clientset   client.Clientset
	PagerConfig *PagerConfig
	Filters     []*meta.RESTMapping
}

// Saver is a snapshot saver.
type Saver struct {
	dynamicClient   dynamic.Interface
	saveConfig      SaveConfig
	patchMetaSchema *patch.PatchMetaFromOpenAPI3

	clock clock.PassiveClock
}

// NewSaver creates a new snapshot saver.
func NewSaver(saveConfig SaveConfig) (*Saver, error) {
	dynamicClient, err := saveConfig.Clientset.ToDynamicClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
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
		patchMetaSchema: patchMetaSchema,
		dynamicClient:   dynamicClient,
		clock:           clock.RealClock{},
	}, nil
}

// Save saves the snapshot of cluster
func (s *Saver) Save(ctx context.Context, encoder *yaml.Encoder, tracks map[*meta.RESTMapping]*TrackData) error {
	logger := log.FromContext(ctx)

	if tracks != nil {
		for _, rm := range s.saveConfig.Filters {
			tracks[rm] = &TrackData{
				Data: map[log.ObjectRef]json.RawMessage{},
			}
		}
	}

	startTime := time.Now()
	totalCounter := 0
	for _, rm := range s.saveConfig.Filters {
		gvr := rm.Resource
		nri := s.dynamicClient.Resource(gvr)
		logger := logger.With("resource", gvr.Resource)

		start := time.Now()
		page := 0

		latestResourceVersion := ""
		listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			var list runtime.Object
			var err error
			page++
			logger := logger.With("page", page, "limit", opts.Limit)
			logger.Debug("Listing resource")
			err = retry.OnError(retry.DefaultBackoff, retriable, func() error {
				l, err := nri.List(ctx, opts)
				if err != nil {
					logger.Error("failed to list resource", err)
				} else {
					list = l
					latestResourceVersion = l.GetResourceVersion()
				}
				return err
			})
			return list, err
		})

		pagerConfig := s.saveConfig.PagerConfig

		if pagerConfig != nil {
			if pagerConfig.PageSize > 0 {
				listPager.PageSize = pagerConfig.PageSize
			}
			if pagerConfig.PageBufferSize > 0 {
				listPager.PageBufferSize = pagerConfig.PageBufferSize
			}
		}

		var track *TrackData
		if tracks != nil {
			track = tracks[rm]
			if track == nil {
				track = &TrackData{
					Data: map[log.ObjectRef]json.RawMessage{},
				}
				tracks[rm] = track
			}
		}
		count := 0
		if err := listPager.EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
			if o, ok := obj.(metav1.Object); ok {
				o.SetResourceVersion("")
				if track != nil {
					track.Data[log.KObj(o)], _ = json.Marshal(o)
				}
			}
			count++
			return encoder.Encode(obj)
		}); err != nil {
			return fmt.Errorf("failed to list resource %q: %w", gvr.Resource, err)
		}

		if track != nil {
			track.ResourceVersion = latestResourceVersion
		}
		logger.Debug("Listed resource",
			"counter", count,
			"elapsed", time.Since(start),
		)
		totalCounter += count
	}

	if tracks == nil {
		if totalCounter == 0 {
			return ErrNotHandled
		}
	}
	logger.Info("Saved resources",
		"counter", totalCounter,
		"elapsed", time.Since(startTime),
	)

	return nil
}

// Record records the snapshot of cluster.
func (s *Saver) Record(ctx context.Context, encoder *yaml.Encoder, tracks map[*meta.RESTMapping]*TrackData) error {
	logger := log.FromContext(ctx)

	startTime := time.Now()

	que := queue.NewQueue[*recording.ResourcePatch]()

	for rm, track := range tracks {
		gvr := rm.Resource
		patchMeta, err := s.patchMetaSchema.Lookup(gvr)
		if err != nil {
			return fmt.Errorf("failed to lookup patch meta: %w", err)
		}
		nri := s.dynamicClient.Resource(gvr)

		w, err := nri.Watch(ctx, metav1.ListOptions{
			ResourceVersion: track.ResourceVersion,
		})
		if err != nil {
			return fmt.Errorf("failed to watch resource %q: %w", gvr.Resource, err)
		}

		go s.buildResourcePatchWorker(ctx, w, que, patchMeta, gvr, startTime, track.Data)
	}

	h := heap.NewHeap[time.Duration, *recording.ResourcePatch]()
	for ctx.Err() == nil {
		resourcePatch, ok := que.GetOrWaitWithDone(ctx.Done())
		if !ok {
			break
		}

		h.Push(resourcePatch.DurationNanosecond, resourcePatch)

		// Tolerate events that are out of order over a period of time
		if h.Len() >= 1024 {
			_, rp, _ := h.Pop()

			err := encoder.Encode(rp)
			if err != nil {
				logger.Warn("Failed to encode resource patch", "err", err)
			}
		}
	}

	// Flush the remaining events
	for {
		_, rp, ok := h.Pop()
		if !ok {
			break
		}
		err := encoder.Encode(rp)
		if err != nil {
			logger.Warn("Failed to encode resource patch", "err", err)
		}
	}

	return nil
}

func (s *Saver) buildResourcePatchWorker(ctx context.Context, w watch.Interface, que queue.Queue[*recording.ResourcePatch], patchMeta strategicpatch.LookupPatchMeta, gvr schema.GroupVersionResource, startTime time.Time, track map[log.ObjectRef]json.RawMessage) {
	logger := log.FromContext(ctx)
	logger = logger.With("resource", gvr.Resource)
	ch := w.ResultChan()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}

			resourcePatch, err := s.buildResourcePatch(ctx, event, patchMeta, gvr, startTime, track)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				logger.Warn("Failed to generate resource patch", "err", err)
				continue
			}
			que.Add(resourcePatch)
		}
	}
}

func retriable(err error) bool {
	return apierrors.IsInternalError(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		net.IsConnectionRefused(err)
}

type TrackData struct {
	Data            map[log.ObjectRef]json.RawMessage
	ResourceVersion string
}

func (s *Saver) buildResourcePatch(ctx context.Context, event watch.Event, patchMeta strategicpatch.LookupPatchMeta, gvr schema.GroupVersionResource, startTime time.Time, track map[log.ObjectRef]json.RawMessage) (*recording.ResourcePatch, error) {
	switch obj := event.Object.(type) {
	default:
		return nil, fmt.Errorf("unknown object type: %T", obj)
	case *metav1.Status:
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		return nil, fmt.Errorf("error status: %s: %s", obj.Reason, obj.Message)
	case metav1.Object:
		rp := recording.ResourcePatch{}
		rp.TypeMeta = recording.ResourcePatchType
		rp.SetTargetGroupVersionResource(gvr)
		rp.SetTargetName(obj.GetName(), obj.GetNamespace())
		now := s.clock.Now()
		rp.SetDuration(now.Sub(startTime))

		switch event.Type {
		case watch.Added, watch.Modified:
			obj.SetResourceVersion("")

			err := rp.SetContent(obj, track, patchMeta)
			if err != nil {
				return nil, err
			}
			return &rp, nil
		case watch.Deleted:
			rp.SetDelete(obj, track)

			return &rp, nil
		default:
			return nil, fmt.Errorf("unknown event type: %s", event.Type)
		}
	}
}
