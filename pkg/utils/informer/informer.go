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
	"fmt"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/pager"
	"k8s.io/client-go/util/watchlist"

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
	if events == nil {
		return fmt.Errorf("events channel is nil")
	}

	fun := func(obj runtime.Object) error {
		if ok, err := opt.filter(obj); err != nil {
			return err
		} else if !ok {
			return nil
		}
		events <- Event[T]{Type: Sync, Object: obj.(T)}
		return nil
	}

	if opt.EnableStreamWatch {
		if watchListOptions, hasWatchListOptionsPrepared, watchListOptionsErr := watchlist.PrepareWatchListOptionsFromListOptions(opt.toListOptions()); watchListOptionsErr != nil {
			log.FromContext(ctx).Error("Failed preparing watchlist options, falling back to the standard LIST semantics", watchListOptionsErr)
		} else if hasWatchListOptionsPrepared {
			watcher, err := i.WatchFunc(ctx, watchListOptions)
			if err != nil {
				return err
			}
			return handleWatchList(ctx, watcher, fun)
		}
	}

	if !opt.EnableListPager {
		list, err := i.ListFunc(ctx, opt.toListOptions())
		if err != nil {
			return err
		}
		return meta.EachListItem(list, fun)
	}

	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return i.ListFunc(ctx, opts)
	})

	err := listPager.EachListItem(ctx, opt.toListOptions(), fun)
	if err != nil {
		return err
	}
	return nil
}

func (i *Informer[T, L]) listWatch(ctx context.Context) *cache.ListWatch {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return i.ListFunc(ctx, opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return i.WatchFunc(ctx, opts)
		},
	}
}

// WatchWithLazyCache starts a goroutine that watches the resource and sends events to the events channel.
func (i *Informer[T, L]) WatchWithLazyCache(ctx context.Context, opt Option, events chan<- Event[T]) (Getter[T], error) {
	lw := i.listWatch(ctx)

	dummyCtx, dummyCancel := context.WithCancel(ctx)

	dummyInformer := newDummyInformer(ctx, lw, opt, events)
	go dummyInformer.Run(dummyCtx.Done())

	l := &lazyGetter[T]{
		initStore: func() cache.Store {
			dummyCancel()

			c, controller := newCacheInformer(ctx, lw, opt, events)
			go controller.Run(ctx.Done())
			return c
		},
	}
	return l, nil
}

// WatchWithCache starts a goroutine that watches the resource and sends events to the events channel.
func (i *Informer[T, L]) WatchWithCache(ctx context.Context, opt Option, events chan<- Event[T]) (Getter[T], error) {
	store, controller := newCacheInformer[T](ctx, i.listWatch(ctx), opt, events)
	go controller.Run(ctx.Done())

	g := &getter[T]{store: store}
	return g, nil
}

func newCacheInformer[T runtime.Object](ctx context.Context, listWatch cache.ListerWatcherWithContext, opt Option, events chan<- Event[T]) (cache.Store, cache.Controller) {
	var t T
	eventHandler := cache.ResourceEventHandlerFuncs{}
	if events != nil {
		eventHandler = cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				if ok, err := opt.filter(obj); err != nil {
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Added, Object: obj.(T)}
			},
			UpdateFunc: func(oldObj, newObj any) {
				if ok, err := opt.filter(newObj); err != nil {
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Modified, Object: newObj.(T)}
			},
			DeleteFunc: func(obj any) {
				if ok, err := opt.filter(obj); err != nil {
					return
				} else if !ok {
					return
				}
				events <- Event[T]{Type: Deleted, Object: obj.(T)}
			},
		}
	}

	store, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opt.setup(&opts)
				return listWatch.ListWithContext(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opt.setup(&opts)
				return listWatch.WatchWithContext(ctx, opts)
			},
		},
		ObjectType: objType(t),
		Handler:    eventHandler,
	})

	return store, controller
}

// Watch starts a goroutine that watches the resource and sends events to the events channel.
func (i *Informer[T, L]) Watch(ctx context.Context, opt Option, events chan<- Event[T]) error {
	if events == nil {
		return fmt.Errorf("events channel is nil")
	}

	informer := newDummyInformer(ctx, i.listWatch(ctx), opt, events)
	go informer.Run(ctx.Done())

	return nil
}

func newDummyInformer[T runtime.Object](ctx context.Context, listWatch cache.ListerWatcherWithContext, opt Option, events chan<- Event[T]) *cache.Reflector {
	var t T
	informer := cache.NewReflectorWithOptions(
		&cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opt.setup(&opts)
				return listWatch.ListWithContext(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opt.setup(&opts)
				return listWatch.WatchWithContext(ctx, opts)
			},
		},
		objType(t),
		dummyCache(events, opt),
		cache.ReflectorOptions{},
	)
	return informer
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

type lazyGetter[T runtime.Object] struct {
	store cache.Store

	onceInit  sync.Once
	initStore func() cache.Store
}

func (l *lazyGetter[T]) init() {
	l.store = l.initStore()
}

func (l *lazyGetter[T]) Get(name string) (t T, exists bool) {
	l.onceInit.Do(l.init)
	obj, exists, err := l.store.GetByKey(name)
	if err != nil {
		return t, false
	}
	if !exists {
		return t, false
	}
	return obj.(T), true
}

func (l *lazyGetter[T]) GetWithNamespace(name, namespace string) (t T, exists bool) {
	return l.Get(namespace + "/" + name)
}

func (l *lazyGetter[T]) List() (list []T) {
	l.onceInit.Do(l.init)
	for _, obj := range l.store.List() {
		list = append(list, obj.(T))
	}
	return list
}

func objType(expectedType runtime.Object) runtime.Object {
	switch expectedType.(type) {
	default:
		return expectedType
	case *unstructured.Unstructured:
		var obj unstructured.Unstructured
		return &obj
	}
}

// handleWatchList copy from watchListClient feature with some modifications to allow streaming handle.
// Note that this function will close the passed watch.
func handleWatchList(ctx context.Context, w watch.Interface, eventsFunc func(obj runtime.Object) error) error {
	defer w.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-w.ResultChan():
			if !ok {
				return fmt.Errorf("unexpected watch close")
			}
			if event.Type == watch.Error {
				return apierrors.FromObject(event.Object)
			}
			meta, err := meta.Accessor(event.Object)
			if err != nil {
				return fmt.Errorf("failed to parse watch event: %#v", event)
			}

			switch event.Type {
			case watch.Added:
				if err = eventsFunc(event.Object); err != nil {
					return err
				}
			case watch.Bookmark:
				if meta.GetAnnotations()[metav1.InitialEventsAnnotationKey] == "true" {
					base64EncodedInitialEventsListBlueprint := meta.GetAnnotations()[metav1.InitialEventsListBlueprintAnnotationKey]
					if len(base64EncodedInitialEventsListBlueprint) == 0 {
						return fmt.Errorf("%q annotation is missing content", metav1.InitialEventsListBlueprintAnnotationKey)
					}
					return nil
				}
			default:
				return fmt.Errorf("unexpected watch event %#v, expected to only receive watch.Added and watch.Bookmark events", event)
			}
		}
	}
}
