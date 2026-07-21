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

package controllers

import (
	"fmt"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type workloadObjectGetter[T runtime.Object] struct {
	items map[string]T
}

func (g workloadObjectGetter[T]) Get(name string) (T, bool) {
	item, ok := g.items[name]
	return item, ok
}

func (g workloadObjectGetter[T]) GetWithNamespace(name, namespace string) (T, bool) {
	return g.Get(namespace + "/" + name)
}

func (g workloadObjectGetter[T]) List() []T {
	items := make([]T, 0, len(g.items))
	for _, item := range g.items {
		items = append(items, item)
	}
	return items
}

func workloadControllerOwner(apiVersion, kind, name, uid string) metav1.OwnerReference {
	controller := true
	return metav1.OwnerReference{APIVersion: apiVersion, Kind: kind, Name: name, UID: types.UID(uid), Controller: &controller}
}

func testWorkloadResolver(replicaSets workloadObjectGetter[*appsv1.ReplicaSet], jobs workloadObjectGetter[*batchv1.Job]) *workloadResolver {
	return newWorkloadResolver(
		replicaSets,
		workloadObjectGetter[*appsv1.Deployment]{items: map[string]*appsv1.Deployment{}},
		workloadObjectGetter[*appsv1.StatefulSet]{items: map[string]*appsv1.StatefulSet{}},
		workloadObjectGetter[*appsv1.DaemonSet]{items: map[string]*appsv1.DaemonSet{}},
		jobs,
		workloadObjectGetter[*batchv1.CronJob]{items: map[string]*batchv1.CronJob{}},
	)
}

func metricPod(name, uid string, created time.Time, owner *metav1.OwnerReference) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(uid), CreationTimestamp: metav1.NewTime(created)},
		Spec: corev1.PodSpec{
			NodeName:   "node",
			Containers: []corev1.Container{{Name: "container"}},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if owner != nil {
		pod.OwnerReferences = []metav1.OwnerReference{*owner}
	}
	return pod
}

func TestWorkloadResolverDeploymentRollout(t *testing.T) {
	created := metav1.NewTime(time.Now().Add(-time.Hour))
	deploymentOwner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "Deployment", "deployment", "deployment-uid")
	replicaSets := workloadObjectGetter[*appsv1.ReplicaSet]{items: map[string]*appsv1.ReplicaSet{
		"default/old": {ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default", OwnerReferences: []metav1.OwnerReference{deploymentOwner}}},
		"default/new": {ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default", OwnerReferences: []metav1.OwnerReference{deploymentOwner}}},
	}}
	resolver := newWorkloadResolver(
		replicaSets,
		workloadObjectGetter[*appsv1.Deployment]{items: map[string]*appsv1.Deployment{
			"default/deployment": {ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "default", CreationTimestamp: created}},
		}},
		workloadObjectGetter[*appsv1.StatefulSet]{items: map[string]*appsv1.StatefulSet{}},
		workloadObjectGetter[*appsv1.DaemonSet]{items: map[string]*appsv1.DaemonSet{}},
		workloadObjectGetter[*batchv1.Job]{items: map[string]*batchv1.Job{}},
		workloadObjectGetter[*batchv1.CronJob]{items: map[string]*batchv1.CronJob{}},
	)
	now := time.Now()
	oldOwner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "ReplicaSet", "old", "old-uid")
	newOwner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "ReplicaSet", "new", "new-uid")
	pods := []*corev1.Pod{
		metricPod("new-1", "uid-3", now.Add(2*time.Second), &newOwner),
		metricPod("old-1", "uid-1", now, &oldOwner),
		metricPod("new-0", "uid-2", now.Add(time.Second), &newOwner),
	}
	for _, pod := range pods {
		resolver.updatePod(pod)
	}
	for index, name := range []string{"old-1", "new-0", "new-1"} {
		var pod *corev1.Pod
		for _, candidate := range pods {
			if candidate.Name == name {
				pod = candidate
			}
		}
		workload := resolver.ResolveWorkload(pod)
		if workload.Kind != "Deployment" || workload.Name != "deployment" || workload.UID != "deployment-uid" {
			t.Fatalf("unexpected workload identity for %s: %#v", name, workload)
		}
		if workload.Replicas != 3 || workload.PodIndex != int64(index) {
			t.Fatalf("unexpected membership for %s: replicas=%d index=%d", name, workload.Replicas, workload.PodIndex)
		}
		if !workload.CreationTimestamp.Equal(&created) {
			t.Fatalf("expected Deployment creation timestamp for %s, got %v", name, workload.CreationTimestamp)
		}
	}

	resolver.deletePod(pods[0])
	if workload := resolver.ResolveWorkload(pods[1]); workload.Replicas != 2 || workload.PodIndex != 0 {
		t.Fatalf("unexpected workload after scale down: %#v", workload)
	}
}

func TestWorkloadResolverKindsAndInactivePods(t *testing.T) {
	cronJobOwner := workloadControllerOwner(batchv1.SchemeGroupVersion.String(), "CronJob", "cron", "cron-uid")
	jobs := workloadObjectGetter[*batchv1.Job]{items: map[string]*batchv1.Job{
		"default/job": {ObjectMeta: metav1.ObjectMeta{Name: "job", Namespace: "default", OwnerReferences: []metav1.OwnerReference{cronJobOwner}}},
	}}
	resolver := testWorkloadResolver(workloadObjectGetter[*appsv1.ReplicaSet]{items: map[string]*appsv1.ReplicaSet{}}, jobs)
	now := time.Now()
	tests := []struct {
		name  string
		owner *metav1.OwnerReference
		kind  string
	}{
		{name: "cron-pod", owner: ptrOwner(workloadControllerOwner(batchv1.SchemeGroupVersion.String(), "Job", "job", "job-uid")), kind: "CronJob"},
		{name: "stateful-pod", owner: ptrOwner(workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "StatefulSet", "stateful", "stateful-uid")), kind: "StatefulSet"},
		{name: "daemon-pod", owner: ptrOwner(workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "DaemonSet", "daemon", "daemon-uid")), kind: "DaemonSet"},
		{name: "unknown-pod", owner: ptrOwner(workloadControllerOwner("example.io/v1", "Example", "example", "example-uid")), kind: "Example"},
		{name: "bare-pod", kind: "Pod"},
	}
	for i, test := range tests {
		pod := metricPod(test.name, fmt.Sprintf("uid-%d", i), now.Add(time.Duration(i)*time.Second), test.owner)
		resolver.updatePod(pod)
		if workload := resolver.ResolveWorkload(pod); workload.Kind != test.kind {
			t.Fatalf("expected %s for %s, got %#v", test.kind, test.name, workload)
		}
	}

	active := metricPod("active", "active", now, nil)
	terminal := metricPod("terminal", "terminal", now, nil)
	terminal.Status.Phase = corev1.PodSucceeded
	statefulOwner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "StatefulSet", "deleting-test", "deleting-test-uid")
	statefulActive := metricPod("stateful-active", "stateful-active", now, &statefulOwner)
	statefulDeleting := metricPod("stateful-deleting", "stateful-deleting", now, &statefulOwner)
	deletionTime := metav1.NewTime(now)
	statefulDeleting.DeletionTimestamp = &deletionTime
	resolver.updatePod(active)
	resolver.updatePod(terminal)
	resolver.updatePod(statefulActive)
	resolver.updatePod(statefulDeleting)
	if workload := resolver.ResolveWorkload(terminal); workload.Kind != "Pod" || workload.Replicas != 1 {
		t.Fatalf("terminal pod should use compatibility workload, got %#v", workload)
	}
	if workload := resolver.ResolveWorkload(statefulActive); workload.Replicas != 1 {
		t.Fatalf("deleting pod should not count toward workload replicas, got %#v", workload)
	}
}

func ptrOwner(owner metav1.OwnerReference) *metav1.OwnerReference {
	return &owner
}

func TestWorkloadResolverConcurrentUpdates(t *testing.T) {
	resolver := testWorkloadResolver(workloadObjectGetter[*appsv1.ReplicaSet]{items: map[string]*appsv1.ReplicaSet{}}, workloadObjectGetter[*batchv1.Job]{items: map[string]*batchv1.Job{}})
	owner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "StatefulSet", "stateful", "stateful-uid")
	pods := make([]*corev1.Pod, 100)
	for i := range pods {
		pods[i] = metricPod(fmt.Sprintf("pod-%03d", i), fmt.Sprintf("uid-%03d", i), time.Now().Add(time.Duration(i)*time.Second), &owner)
		resolver.updatePod(pods[i])
	}

	var wg sync.WaitGroup
	for i := range pods {
		wg.Add(2)
		go func(pod *corev1.Pod) {
			defer wg.Done()
			resolver.ResolveWorkload(pod)
		}(pods[i])
		go func(pod *corev1.Pod) {
			defer wg.Done()
			resolver.updatePod(pod)
		}(pods[i])
	}
	wg.Wait()
	if workload := resolver.ResolveWorkload(pods[0]); workload.Replicas != int64(len(pods)) {
		t.Fatalf("expected %d replicas, got %d", len(pods), workload.Replicas)
	}
}

func BenchmarkWorkloadResolver50000Replicas(b *testing.B) {
	resolver := testWorkloadResolver(workloadObjectGetter[*appsv1.ReplicaSet]{items: map[string]*appsv1.ReplicaSet{}}, workloadObjectGetter[*batchv1.Job]{items: map[string]*batchv1.Job{}})
	owner := workloadControllerOwner(appsv1.SchemeGroupVersion.String(), "StatefulSet", "stateful", "stateful-uid")
	var pod *corev1.Pod
	for i := 0; i < 50000; i++ {
		pod = metricPod(fmt.Sprintf("pod-%03d", i), fmt.Sprintf("uid-%03d", i), time.Unix(int64(i), 0), &owner)
		resolver.updatePod(pod)
	}
	resolver.ResolveWorkload(pod)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.ResolveWorkload(pod)
	}
}
