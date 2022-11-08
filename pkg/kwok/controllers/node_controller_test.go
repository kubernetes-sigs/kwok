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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNodeController(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node0",
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "10.0.0.0",
					},
				},
				Capacity: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other-node",
			},
			Status: corev1.NodeStatus{},
		},
	)

	nodeSelectorFunc := func(node *corev1.Node) bool {
		return strings.HasPrefix(node.Name, "node")
	}
	nodes, err := NewNodeController(NodeControllerConfig{
		ClientSet:                clientset,
		NodeIP:                   "10.0.0.1",
		NodeSelectorFunc:         nodeSelectorFunc,
		NodeStatusTemplate:       templates.DefaultNodeStatusTemplate,
		NodeHeartbeatTemplate:    templates.DefaultNodeHeartbeatTemplate,
		FuncMap:                  funcMap,
		NodeHeartbeatInterval:    1 * time.Second,
		NodeHeartbeatParallelism: 2,
		LockNodeParallelism:      2,
		Logger:                   testingLogger{t},
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new nodes controller error: %v", err))
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	err = nodes.Start(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to start nodes controller: %w", err))
	}

	time.Sleep(2 * time.Second)

	node0, err := clientset.CoreV1().Nodes().Get(ctx, "node0", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get node0: %w", err))
	}
	if node0.Status.Allocatable[corev1.ResourceCPU] != resource.MustParse("4") {
		t.Fatal(fmt.Errorf("node0 want 4 cpu, got %v", node0.Status.Allocatable[corev1.ResourceCPU]))
	}

	node1 := node0.DeepCopy()
	node1.Name = "node1"
	node1.Status.Allocatable[corev1.ResourceCPU] = resource.MustParse("16")
	_, err = clientset.CoreV1().Nodes().Create(ctx, node1, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create node1: %w", err))
	}
	time.Sleep(2 * time.Second)

	nodeSize := nodes.Size()
	if nodeSize != 2 {
		t.Fatal(fmt.Errorf("want 2 nodes, got %d", nodeSize))
	}

	node1, err = clientset.CoreV1().Nodes().Get(ctx, "node1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get node1: %w", err))
	}
	if node1.Status.Allocatable[corev1.ResourceCPU] != resource.MustParse("16") {
		t.Fatal(fmt.Errorf("node1 want 8 cpu, got %v", node1.Status.Allocatable[corev1.ResourceCPU]))
	}

	list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to list nodes: %w", err))
	}
	for _, node := range list.Items {
		if nodeSelectorFunc(&node) {
			if node.Status.Phase != corev1.NodeRunning {
				t.Fatal(fmt.Errorf("want node %s to be running, got %s", node.Name, node.Status.Phase))
			}
		} else {
			if node.Status.Phase == corev1.NodeRunning {
				t.Fatal(fmt.Errorf("want node %s to be not running, got %s", node.Name, node.Status.Phase))
			}
		}
	}
}
