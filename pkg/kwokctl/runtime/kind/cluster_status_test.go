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

package kind

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInspectPodReadiness(t *testing.T) {
	ctx := context.Background()
	pod := &corev1.Pod{
		Name:      "kwok-controller-kwok-test-control-plane",
		Namespace: metav1.NamespaceSystem,
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "kwok-controller",
					// The kubelet keeps the container not ready while its readiness probe fails.
					Ready: false,
				},
			},
		},
	}
	client := fake.NewClientset(pod)

	check := func(wantReady, wantRunning, wantExist bool) {
		t.Helper()
		ready, running, exist, err := inspectPod(ctx, client, pod.Name)
		if err != nil {
			t.Fatal(err)
		}
		if ready != wantReady || running != wantRunning || exist != wantExist {
			t.Fatalf("inspectPod() = ready %v, running %v, exist %v, want %v, %v, %v",
				ready, running, exist, wantReady, wantRunning, wantExist)
		}
	}

	check(false, true, true)

	// The readiness probe succeeds.
	pod.Status.ContainerStatuses[0].Ready = true
	_, err := client.CoreV1().Pods(metav1.NamespaceSystem).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	check(true, true, true)

	// The readiness probe fails again.
	pod.Status.ContainerStatuses[0].Ready = false
	_, err = client.CoreV1().Pods(metav1.NamespaceSystem).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	check(false, true, true)

	pod.Status.Phase = corev1.PodPending
	_, err = client.CoreV1().Pods(metav1.NamespaceSystem).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	check(false, false, true)

	ready, _, exist, err := inspectPod(ctx, client, "missing")
	if err != nil || ready || exist {
		t.Fatalf("inspectPod() for a missing pod = ready %v, exist %v, err %v", ready, exist, err)
	}
}
