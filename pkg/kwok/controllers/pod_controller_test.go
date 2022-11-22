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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/kwok/controllers/templates"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"
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

	nodeInfoGetFunc := func(nodeName string) *nodeInfo {
		if nodeHasFunc(nodeName) {
			return &nodeInfo{}
		}
		return nil
	}

	annotationSelector, _ := labels.Parse("fake=custom")
	pods, err := NewPodController(PodControllerConfig{
		ClientSet:                             clientset,
		NodeIP:                                "10.0.0.1",
		CIDR:                                  "10.0.0.1/24",
		DisregardStatusWithAnnotationSelector: annotationSelector.String(),
		PodStatusTemplate:                     templates.DefaultPodStatusTemplate,
		NodeHasFunc:                           nodeHasFunc,
		NodeInfoGetFunc:                       nodeInfoGetFunc,
		FuncMap:                               funcMap,
		LockPodParallelism:                    2,
		DeletePodParallelism:                  2,
		Logger:                                testingLogger{t},
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new pods controller error: %w", err))
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	err = pods.Start(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("start pods controller error: %w", err))
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
		t.Fatal(fmt.Errorf("get pod1 error: %w", err))
	}
	pod1.Annotations = map[string]string{
		"fake": "custom",
	}
	pod1.Status.Reason = "custom"
	clientset.CoreV1().Pods("default").Update(ctx, pod1, metav1.UpdateOptions{})

	pod1, err = clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %w", err))
	}
	if pod1.Status.Reason != "custom" {
		t.Fatal(fmt.Errorf("pod1 status reason not custom"))
	}

	time.Sleep(2 * time.Second)

	list, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list pods error: %w", err))
	}

	if len(list.Items) != 3 {
		t.Fatal(fmt.Errorf("want 3 pods, got %d", len(list.Items)))
	}

	pod := list.Items[0]
	now := metav1.Now()
	pod.DeletionTimestamp = &now
	_, err = clientset.CoreV1().Pods("default").Update(ctx, &pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("delete pod error: %w", err))
	}

	time.Sleep(2 * time.Second)
	list, err = clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list pods error: %w", err))
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

func TestPodControllerIPPool(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	nodeHasFunc := func(nodeName string) bool {
		return strings.HasPrefix(nodeName, "node")
	}

	node1PodCIDR := "10.0.1.1/24"
	node1PodNet, _ := parseCIDR(node1PodCIDR)
	node1Info := &nodeInfo{
		CidrIPNet: node1PodNet,
		IPPool:    newIPPool(node1PodNet),
	}
	nodeInfoGetFunc := func(nodeName string) *nodeInfo {
		if nodeName == "node0" {
			return &nodeInfo{}
		}
		return node1Info
	}

	podCIDR := "10.0.0.1/24"
	pods, err := NewPodController(PodControllerConfig{
		ClientSet:            clientset,
		NodeIP:               "10.0.0.1",
		CIDR:                 podCIDR,
		PodStatusTemplate:    templates.DefaultPodStatusTemplate,
		NodeHasFunc:          nodeHasFunc,
		NodeInfoGetFunc:      nodeInfoGetFunc,
		FuncMap:              funcMap,
		LockPodParallelism:   2,
		DeletePodParallelism: 2,
		Logger:               testingLogger{t},
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new pods controller error: %w", err))
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	err = pods.Start(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("start pods controller error: %w", err))
	}

	var genPod = func(podName, nodeName string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              podName,
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
				NodeName: nodeName,
			},
		}
	}

	clientset.CoreV1().Pods("default").Create(ctx, genPod("pod0", "node0"), metav1.CreateOptions{})

	// sleep 2 seconds to wait for pod0 to be assigned an IP
	time.Sleep(2 * time.Second)

	pod0, err := clientset.CoreV1().Pods("default").Get(ctx, "pod0", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod0 error: %w", err))
	}

	// check if pod0 ip is in default ip cidr
	pod0IP := pod0.Status.PodIP
	if pod0IP == "" {
		t.Fatal(fmt.Errorf("want pod %s to be assign an IP, but got nothing", pod0.Name))
	}
	if !pods.ipPool.InUsed(pod0IP) {
		t.Fatal(fmt.Errorf("want pod %s ip in %s, but got %s", pod0.Name, podCIDR, pod0IP))
	}

	clientset.CoreV1().Pods("default").Create(ctx, genPod("pod1", "node1"), metav1.CreateOptions{})

	// sleep 2 seconds to wait for pod0 to be assigned an IP
	time.Sleep(2 * time.Second)

	pod1, err := clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %w", err))
	}

	// check if pod1 ip is in node pod cidr
	pod1IP := pod1.Status.PodIP
	if pod1IP == "" {
		t.Fatal(fmt.Errorf("want pod %s to be assign an IP, but got nothing", pod1.Name))
	}
	if !node1Info.IPPool.InUsed(pod1IP) {
		t.Fatal(fmt.Errorf("want pod %s ip in %s, but got %s", pod1.Name, node1PodCIDR, pod1IP))
	}

	clientset.CoreV1().Pods("default").Delete(ctx, "pod0", metav1.DeleteOptions{})
	// sleep 2 seconds to wait for pod0 to be deleted
	time.Sleep(2 * time.Second)
	if pods.ipPool.InUsed(pod0IP) {
		t.Fatal(fmt.Errorf("want pod0 ip to be reclaimed, but got %s in use", pod0IP))
	}

	clientset.CoreV1().Pods("default").Delete(ctx, "pod1", metav1.DeleteOptions{})
	// sleep 2 seconds to wait for pod1 to be deleted
	time.Sleep(2 * time.Second)
	if node1Info.IPPool.InUsed(pod1IP) {
		t.Fatal(fmt.Errorf("want pod1 ip to be reclaimed, but got %s in use", pod1IP))
	}
}
