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

package resources

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Syncer is an interface for syncing resources.
type Syncer[T runtime.Object, L runtime.Object] interface {
	UpdateStatus(ctx context.Context, obj T, opts metav1.UpdateOptions) (T, error)
	List(ctx context.Context, opts metav1.ListOptions) (L, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// ConvertFunc converts a list of resources to a single resource.
type ConvertFunc[O any, T runtime.Object, S ~[]T] func(objs S) O

// NewDynamicGetter returns a new Getter that returns the latest list of resources.
func NewDynamicGetter[O any, T runtime.Object, L runtime.Object](syncer Syncer[T, L], convertFunc ConvertFunc[O, T, []T]) DynamicGetter[O] {
	syncCh := make(chan struct{}, 1)
	syncCh <- struct{}{}
	getter := &dynamicGetter[O, T, L]{
		syncCh:      syncCh,
		syncer:      syncer,
		convertFunc: convertFunc,
	}

	return struct {
		Getter[O]
		Starter
		Synced
	}{
		Getter:  withCache[O](getter),
		Starter: getter,
		Synced:  getter,
	}
}

type dynamicGetter[O any, T runtime.Object, L runtime.Object] struct {
	syncer Syncer[T, L]
	syncCh chan struct{}

	convertFunc ConvertFunc[O, T, []T]

	store      cache.Store
	controller cache.Controller
}

func (c *dynamicGetter[O, T, L]) Start(ctx context.Context) error {
	var t T
	store, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return c.syncer.List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return c.syncer.Watch(ctx, options)
			},
		},
		t,
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				c.sync()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				c.sync()
			},
			DeleteFunc: func(obj interface{}) {
				c.sync()
			},
		},
	)

	c.store = store
	c.controller = controller
	go controller.Run(ctx.Done())
	return store.Resync()
}

func (c *dynamicGetter[O, T, L]) Get() O {
	list := c.store.List()
	currentList := make([]T, 0, len(list))
	for _, obj := range list {
		currentList = append(currentList, obj.(T))
	}

	data := c.convertFunc(currentList)
	return data
}

func (c *dynamicGetter[O, T, L]) Version() string {
	return c.controller.LastSyncResourceVersion()
}

func (c *dynamicGetter[O, T, L]) Sync() <-chan struct{} {
	return c.syncCh
}

func (c *dynamicGetter[O, T, L]) sync() {
	select {
	case c.syncCh <- struct{}{}:
	default:
	}
}
