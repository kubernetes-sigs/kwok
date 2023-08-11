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

package informer

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/pager"

	"sigs.k8s.io/kwok/pkg/log"
)

// Informer is a wrapper around a Get/List/Watch function.
type Informer[T runtime.Object, L runtime.Object] struct {
	ListFunc  func(ctx context.Context, opts metav1.ListOptions) (L, error)
	WatchFunc func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// NewInformer returns a new Informer.
func NewInformer[T runtime.Object, L runtime.Object](lw Watcher[T, L]) *Informer[T, L] {
	return &Informer[T, L]{
		ListFunc:  lw.List,
		WatchFunc: lw.Watch,
	}
}

// Sync sends a sync event for each resource returned by the ListFunc.
func (i *Informer[T, L]) Sync(ctx context.Context, opt Option, events chan<- Event[T]) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return i.ListFunc(ctx, opts)
	})

	err := listPager.EachListItem(ctx, opt.toListOptions(), func(obj runtime.Object) error {
		if ok, err := opt.filter(obj); err != nil {
			return err
		} else if !ok {
			return nil
		}
		events <- Event[T]{Type: Sync, Object: obj.(T)}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// WatchWithCache starts a goroutine that watches the resource and sends events to the events channel.
func (i *Informer[T, L]) WatchWithCache(ctx context.Context, opt Option, events chan<- Event[T]) (Getter[T], error) {
	var t T
	logger := log.FromContext(ctx)
	store, contrtoller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opt.setup(&opts)
				return i.ListFunc(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opt.setup(&opts)
				return i.WatchFunc(ctx, opts)
			},
		},
		t,
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				if ok, err := opt.filter(obj); err != nil {
					logger.Error("filtering object", err)
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Added, Object: obj.(T)}
			},
			UpdateFunc: func(oldObj, newObj any) {
				if ok, err := opt.filter(newObj); err != nil {
					logger.Error("filtering object", err)
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Modified, Object: newObj.(T)}
			},
			DeleteFunc: func(obj any) {
				if ok, err := opt.filter(obj); err != nil {
					logger.Error("filtering object", err)
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Deleted, Object: obj.(T)}
			},
		},
	)

	go contrtoller.Run(ctx.Done())

	g := &getter[T]{store: store}
	return g, nil
}

// Watch starts a goroutine that watches the resource and sends events to the events channel.
func (i *Informer[T, L]) Watch(ctx context.Context, opt Option, events chan<- Event[T]) error {
	var t T
	informer := cache.NewReflectorWithOptions(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opt.setup(&opts)
				return i.ListFunc(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opt.setup(&opts)
				return i.WatchFunc(ctx, opts)
			},
		},
		t,
		dummyCache(events, opt),
		cache.ReflectorOptions{},
	)
	go informer.Run(ctx.Done())
	return nil
}

func dummyCache[T runtime.Object](ch chan<- Event[T], opt Option) cache.Store {
	return &cache.FakeCustomStore{
		AddFunc: func(obj any) error {
			if ok, err := opt.filter(obj); err != nil {
				return err
			} else if !ok {
				return nil
			}
			ch <- Event[T]{Type: Added, Object: obj.(T)}
			return nil
		},
		UpdateFunc: func(obj any) error {
			if ok, err := opt.filter(obj); err != nil {
				return err
			} else if !ok {
				return nil
			}
			ch <- Event[T]{Type: Modified, Object: obj.(T)}
			return nil
		},
		DeleteFunc: func(obj any) error {
			if ok, err := opt.filter(obj); err != nil {
				return err
			} else if !ok {
				return nil
			}
			ch <- Event[T]{Type: Deleted, Object: obj.(T)}
			return nil
		},
		ReplaceFunc: func(list []any, resourceVersion string) error {
			for _, obj := range list {
				if ok, err := opt.filter(obj); err != nil {
					return err
				} else if !ok {
					continue
				}
				ch <- Event[T]{Type: Sync, Object: obj.(T)}
			}
			return nil
		},
		ListFunc: func() []any {
			panic("unreachable")
		},
		ListKeysFunc: func() []string {
			panic("unreachable")
		},
		GetFunc: func(obj any) (item any, exists bool, err error) {
			panic("unreachable")
		},
		GetByKeyFunc: func(key string) (item any, exists bool, err error) {
			panic("unreachable")
		},
		ResyncFunc: func() error {
			return nil
		},
	}
}

// Getter is a wrapper around a cache.Store that provides Get and List methods.
type Getter[T runtime.Object] interface {
	Get(name string) (T, bool)
	GetWithNamespace(name, namespace string) (T, bool)
	List() []T
}

type getter[T runtime.Object] struct {
	store cache.Store
}

func (g *getter[T]) Get(name string) (t T, exists bool) {
	obj, exists, err := g.store.GetByKey(name)
	if err != nil {
		return t, false
	}
	if !exists {
		return t, false
	}
	return obj.(T), true
}

func (g *getter[T]) GetWithNamespace(name, namespace string) (t T, exists bool) {
	return g.Get(namespace + "/" + name)
}

func (g *getter[T]) List() (list []T) {
	for _, obj := range g.store.List() {
		list = append(list, obj.(T))
	}
	return list
}
