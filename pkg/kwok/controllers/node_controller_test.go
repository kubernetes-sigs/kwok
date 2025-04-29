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
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	nodefast "sigs.k8s.io/kwok/kustomize/stage/node/fast"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func TestNodeController(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node0",
				Annotations: map[string]string{
					"node": "true",
				},
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
		return node.Annotations["node"] == "true"
	}

	nodeInit, _ := config.UnmarshalWithType[*internalversion.Stage](nodefast.DefaultNodeInit)
	nodeStages := []*internalversion.Stage{nodeInit}

	lc, _ := lifecycle.NewLifecycle(nodeStages, nil)
	nodes, err := NewNodeController(NodeControllerConfig{
		TypedClient:          clientset,
		NodeIP:               "10.0.0.1",
		Lifecycle:            resources.NewStaticGetter(lc),
		PlayStageParallelism: 2,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new nodes controller error: %w", err))
	}
	ctx := context.Background()
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.LevelDebug))
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	nodeCh := make(chan informer.Event[*corev1.Node], 1)
	nodesCli := clientset.CoreV1().Nodes()
	nodesInformer := informer.NewInformer[*corev1.Node, *corev1.NodeList](nodesCli)
	err = nodesInformer.Watch(ctx, informer.Option{
		AnnotationSelector: "node=true",
	}, nodeCh)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to watch nodes: %w", err))
	}

	err = nodes.Start(ctx, nodeCh)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to start nodes controller: %w", err))
	}

	var node0 *corev1.Node
	err = wait.Poll(ctx, func(ctx context.Context) (done bool, err error) {
		node0, err = clientset.CoreV1().Nodes().Get(ctx, "node0", metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get node0: %w", err)
		}
		if node0.Status.Allocatable[corev1.ResourceCPU] != resource.MustParse("4") {
			return false, fmt.Errorf("node0 want 4 cpu, got %v", node0.Status.Allocatable[corev1.ResourceCPU])
		}
		return true, nil
	}, wait.WithContinueOnError(5))
	if err != nil {
		t.Fatal(err)
	}

	node1 := node0.DeepCopy()
	node1.Name = "node1"
	node1.Status.Allocatable[corev1.ResourceCPU] = resource.MustParse("16")
	_, err = clientset.CoreV1().Nodes().Create(ctx, node1, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to create node1: %w", err))
	}

	node1, err = clientset.CoreV1().Nodes().Get(ctx, "node1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get node1: %w", err))
	}
	if node1.Status.Allocatable[corev1.ResourceCPU] != resource.MustParse("16") {
		t.Fatal(fmt.Errorf("node1 want 8 cpu, got %v", node1.Status.Allocatable[corev1.ResourceCPU]))
	}

	err = wait.Poll(ctx, func(ctx context.Context) (done bool, err error) {
		list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to list nodes: %w", err)
		}
		for i, node := range list.Items {
			if nodeSelectorFunc(&list.Items[i]) {
				if node.Status.Phase != corev1.NodeRunning {
					return false, fmt.Errorf("want node %s to be running, got %s", node.Name, node.Status.Phase)
				}
			} else {
				if node.Status.Phase == corev1.NodeRunning {
					return false, fmt.Errorf("want node %s to be not running, got %s", node.Name, node.Status.Phase)
				}
			}
		}
		return true, nil
	}, wait.WithContinueOnError(5))
	if err != nil {
		t.Fatal(err)
	}
}
