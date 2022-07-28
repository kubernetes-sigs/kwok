/*
Copyright 2022 The Kubernetes Authors.

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

package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/kwok/pkg/kwok-controller/templates"
)

func TestPodController(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod0",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "test-image",
					},
				},
				NodeName: "node0",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "xxxx",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "test-container",
						Image: "test-image",
					},
				},
				NodeName: "xxxx",
			},
		},
	)

	nodeHasFunc := func(nodeName string) bool {
		return strings.HasPrefix(nodeName, "node")
	}
	annotationSelector, _ := labels.Parse("fake=custom")
	pods, err := NewPodController(PodControllerConfig{
		ClientSet:                      clientset,
		NodeIP:                         "10.0.0.1",
		CIDR:                           "10.0.0.1/24",
		StatusCustomAnnotationSelector: annotationSelector.String(),
		PodStatusTemplate:              templates.DefaultPodStatusTemplate,
		NodeHasFunc:                    nodeHasFunc,
		FuncMap:                        funcMap,
		LockPodParallelism:             2,
		DeletePodParallelism:           2,
		Logger:                         testingLogger{t},
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new pods controller error: %v", err))
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer func() {
		cancel()
		time.Sleep(time.Second)
	}()

	err = pods.Start(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("start pods controller error: %v", err))
	}

	clientset.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pod1",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
			NodeName: "node0",
		},
	}, metav1.CreateOptions{})

	pod1, err := clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %v", err))
	}
	pod1.Annotations = map[string]string{
		"fake": "custom",
	}
	pod1.Status.Reason = "custom"
	clientset.CoreV1().Pods("default").Update(ctx, pod1, metav1.UpdateOptions{})

	pod1, err = clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %v", err))
	}
	if pod1.Status.Reason != "custom" {
		t.Fatal(fmt.Errorf("pod1 status reason not custom"))
	}

	time.Sleep(2 * time.Second)

	list, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list pods error: %v", err))
	}

	if len(list.Items) != 3 {
		t.Fatal(fmt.Errorf("want 3 pods, got %d", len(list.Items)))
	}

	pod := list.Items[0]
	now := metav1.Now()
	pod.DeletionTimestamp = &now
	_, err = clientset.CoreV1().Pods("default").Update(ctx, &pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("delete pod error: %v", err))
	}

	time.Sleep(2 * time.Second)
	list, err = clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list pods error: %v", err))
	}
	if len(list.Items) != 2 {
		t.Fatal(fmt.Errorf("want 2 pods, got %d", len(list.Items)))
	}

	for _, pod := range list.Items {
		if nodeHasFunc(pod.Spec.NodeName) {
			if pod.Status.Phase != corev1.PodRunning {
				t.Fatal(fmt.Errorf("want pod %s phase is running, got %s", pod.Name, pod.Status.Phase))
			}
		} else {
			if pod.Status.Phase == corev1.PodRunning {
				t.Fatal(fmt.Errorf("want pod %s phase is not running, got %s", pod.Name, pod.Status.Phase))
			}
		}
	}
}
