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

package server

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
)

type fakeGetter[T any] struct {
	items map[string]T
}

func (g fakeGetter[T]) Get(name string) (T, bool) {
	v, ok := g.items[name]
	return v, ok
}

func (g fakeGetter[T]) GetWithNamespace(name, namespace string) (T, bool) {
	v, ok := g.items[namespace+"/"+name]
	return v, ok
}

func (g fakeGetter[T]) List() []T {
	var out []T
	for _, v := range g.items {
		out = append(out, v)
	}
	return out
}

type fakeDataSource struct {
	pods map[string][]log.ObjectRef
}

func (d fakeDataSource) ListPods(nodeName string) ([]log.ObjectRef, bool) {
	pods, ok := d.pods[nodeName]
	return pods, ok
}

func (d fakeDataSource) ListNodes() []string {
	return nil
}

func (d fakeDataSource) StartedContainersTotal(string) int64 {
	return 0
}

func TestNodeResourceCumulativeUsagePerResource(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-0"}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-0", Namespace: "default"},
		Spec: corev1.PodSpec{
			NodeName:   "node-0",
			Containers: []corev1.Container{{Name: "c"}},
		},
	}
	cpu := resource.MustParse("1")
	gpu := resource.MustParse("100")

	s, err := NewServer(Config{
		ClusterResourceUsages: []*internalversion.ClusterResourceUsage{{
			Spec: internalversion.ClusterResourceUsageSpec{
				Usages: []internalversion.ResourceUsageContainer{{
					Usage: map[string]internalversion.ResourceUsageValue{
						"cpu":            {Value: &cpu},
						"nvidia.com/gpu": {Value: &gpu},
					},
				}},
			},
		}},
		DataSource: fakeDataSource{pods: map[string][]log.ObjectRef{
			"node-0": {{Name: "pod-0", Namespace: "default"}},
		}},
		NodeCacheGetter: fakeGetter[*corev1.Node]{items: map[string]*corev1.Node{"node-0": node}},
		PodCacheGetter:  fakeGetter[*corev1.Pod]{items: map[string]*corev1.Pod{"default/pod-0": pod}},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The first call for each resource only records the start time.
	s.nodeResourceCumulativeUsage("cpu", "node-0")
	s.nodeResourceCumulativeUsage("nvidia.com/gpu", "node-0")

	// Move every recorded start time one second into the past.
	s.cumulativesMut.Lock()
	for k, c := range s.nodeCumulatives {
		c.time = c.time.Add(-time.Second)
		s.nodeCumulatives[k] = c
	}
	s.cumulativesMut.Unlock()

	gotCPU := s.nodeResourceCumulativeUsage("cpu", "node-0")
	gotGPU := s.nodeResourceCumulativeUsage("nvidia.com/gpu", "node-0")

	// Each counter covers at least the one second it was moved back.
	// The upper bounds leave room for a slow test run.
	if gotCPU < 1 || gotCPU >= 2 {
		t.Errorf("cpu: want between 1 and 2 core-seconds, got %v", gotCPU)
	}
	if gotGPU < 100 || gotGPU >= 200 {
		t.Errorf("nvidia.com/gpu: want between 100 and 200, got %v", gotGPU)
	}
}
