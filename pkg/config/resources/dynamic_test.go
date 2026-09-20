/*
Copyright 2026 The Kubernetes Authors.

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
	"fmt"
	"slices"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"sigs.k8s.io/kwok/pkg/utils/wait"
)

type fakeSyncer struct {
	list    *corev1.ConfigMapList
	watcher *watch.FakeWatcher
}

func (s *fakeSyncer) UpdateStatus(_ context.Context, obj *corev1.ConfigMap, _ metav1.UpdateOptions) (*corev1.ConfigMap, error) {
	return obj, nil
}

func (s *fakeSyncer) List(_ context.Context, _ metav1.ListOptions) (*corev1.ConfigMapList, error) {
	return s.list.DeepCopy(), nil
}

func (s *fakeSyncer) Watch(_ context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	// The reflector first tries the watch-list protocol; report it as
	// unsupported so that it falls back to a list followed by a watch.
	if opts.SendInitialEvents != nil && *opts.SendInitialEvents {
		return nil, fmt.Errorf("watch-list is not supported")
	}
	return s.watcher, nil
}

// newConfigMap returns a ConfigMap with the given name and resource version.
func newConfigMap(name, resourceVersion string) *corev1.ConfigMap {
	obj := &corev1.ConfigMap{}
	obj.Name = name
	obj.ResourceVersion = resourceVersion
	return obj
}

func TestDynamicGetterVersionFollowsStore(t *testing.T) {
	ctx := t.Context()

	list := &corev1.ConfigMapList{}
	list.ResourceVersion = "1"
	list.Items = []corev1.ConfigMap{*newConfigMap("a", "1")}

	syncer := &fakeSyncer{
		list:    list,
		watcher: watch.NewFake(),
	}

	getter := NewDynamicGetter[[]string, *corev1.ConfigMap, *corev1.ConfigMapList](syncer, func(objs []*corev1.ConfigMap) []string {
		names := make([]string, 0, len(objs))
		for _, obj := range objs {
			names = append(names, obj.Name)
		}
		slices.Sort(names)
		return names
	})

	err := getter.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	waitForVersionChange := func(from string) {
		t.Helper()
		err := wait.Poll(ctx,
			func(ctx context.Context) (bool, error) {
				return getter.Version() != from, nil
			},
			wait.WithImmediate(),
			wait.WithInterval(10*time.Millisecond),
			wait.WithTimeout(10*time.Second),
		)
		if err != nil {
			t.Fatalf("version did not change from %q: %v", from, err)
		}
	}

	// Once Version has changed the store already contains the change, so
	// the cached Get must reflect it without any further waiting.
	expect := func(want []string) {
		t.Helper()
		got := getter.Get()
		if !slices.Equal(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	waitForVersionChange("0")
	expect([]string{"a"})

	version := getter.Version()
	syncer.watcher.Add(newConfigMap("b", "2"))
	waitForVersionChange(version)
	expect([]string{"a", "b"})

	version = getter.Version()
	modified := newConfigMap("b", "3")
	modified.Labels = map[string]string{"k": "v"}
	syncer.watcher.Modify(modified)
	waitForVersionChange(version)
	expect([]string{"a", "b"})

	version = getter.Version()
	syncer.watcher.Delete(newConfigMap("a", "4"))
	waitForVersionChange(version)
	expect([]string{"b"})

	select {
	case <-getter.Sync():
	default:
		t.Fatal("expected a pending sync notification")
	}
}
