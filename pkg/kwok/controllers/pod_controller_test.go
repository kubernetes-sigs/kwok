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
	"net"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/stages"
)

const (
	defaultNodeIP  = "10.0.0.1"
	defaultPodCIDR = "10.100.0.1/24"

	secondNodeIP  = "172.0.0.1"
	secondPodCIDR = "172.100.0.1/24"
)

func TestPodController(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "node0",
				CreationTimestamp: metav1.Now(),
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "node1",
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.NodeSpec{
				PodCIDR: secondPodCIDR,
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: secondNodeIP,
					},
				},
			},
		},
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
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod-with-host-network",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			Spec: corev1.PodSpec{
				HostNetwork: true,
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
				Name:              "pod-with-node1",
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
				NodeName: "node1",
			},
		},
	)

	nodeGetFunc := func(nodeName string) (*NodeInfo, bool) {
		node, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, false
		}

		nodeIP := defaultNodeIP
		podCIDR := defaultPodCIDR

		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodeIP = addr.Address
				break
			}
		}

		if node.Spec.PodCIDR != "" {
			podCIDR = node.Spec.PodCIDR
		}

		nodeInfo := &NodeInfo{
			HostIPs:  []string{nodeIP},
			PodCIDRs: []string{podCIDR},
		}
		return nodeInfo, true
	}
	nodeHasMetric := func(nodeName string) bool {
		return false
	}
	podStages, _ := NewStagesFromYaml([]byte(stages.DefaultPodStages))
	annotationSelector, _ := labels.Parse("fake=custom")
	pods, err := NewPodController(PodControllerConfig{
		ClientSet:                             clientset,
		NodeIP:                                defaultNodeIP,
		CIDR:                                  defaultPodCIDR,
		DisregardStatusWithAnnotationSelector: annotationSelector.String(),
		Stages:                                podStages,
		NodeGetFunc:                           nodeGetFunc,
		NodeHasMetric:                         nodeHasMetric,
		FuncMap:                               defaultFuncMap,
		PlayStageParallelism:                  2,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new pods controller error: %w", err))
	}

	ctx := context.Background()
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.LevelDebug))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	err = pods.Start(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("start pods controller error: %w", err))
	}

	listNodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list nodes error: %w", err))
	}
	for _, node := range listNodes.Items {
		if nodeInfo, ok := nodeGetFunc(node.Name); ok {
			if node.Spec.PodCIDR != "" {
				if node.Spec.PodCIDR != nodeInfo.PodCIDRs[0] {
					t.Fatal(fmt.Errorf("want node %s podCIDR=%s, got %s", node.Name, node.Spec.PodCIDR, nodeInfo.PodCIDRs[0]))
				}
			} else {
				if defaultPodCIDR != nodeInfo.PodCIDRs[0] {
					t.Fatal(fmt.Errorf("want node %s podCIDR=%s, got %s", node.Name, defaultPodCIDR, nodeInfo.PodCIDRs[0]))
				}
			}

			if len(node.Status.Addresses) != 0 {
				if node.Status.Addresses[0].Address != nodeInfo.HostIPs[0] {
					t.Fatal(fmt.Errorf("want node %s address=%s, got %s", node.Name, node.Status.Addresses[0].Address, nodeInfo.HostIPs[0]))
				}
			} else {
				if defaultNodeIP != nodeInfo.HostIPs[0] {
					t.Fatal(fmt.Errorf("want node %s address=%s, got %s", node.Name, defaultNodeIP, nodeInfo.HostIPs[0]))
				}
			}
		}
	}

	_, err = clientset.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
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
	if err != nil {
		t.Fatal(fmt.Errorf("create pod1 error: %w", err))
	}

	pod1, err := clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %w", err))
	}
	pod1.Annotations = map[string]string{
		"fake": "custom",
	}
	pod1.Status.Reason = "custom"
	_, err = clientset.CoreV1().Pods("default").Update(ctx, pod1, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("update pod1 error: %w", err))
	}

	pod1, err = clientset.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("get pod1 error: %w", err))
	}
	if pod1.Status.Reason != "custom" {
		t.Fatal(fmt.Errorf("pod1 status reason not custom"))
	}

	var list *corev1.PodList
	err = wait.Poll(ctx, func(ctx context.Context) (done bool, err error) {
		list, err = clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("list pods error: %w", err)
		}
		if len(list.Items) != 5 {
			return false, fmt.Errorf("want 5 pods, got %d", len(list.Items))
		}

		for index, pod := range list.Items {
			if nodeInfo, ok := nodeGetFunc(pod.Spec.NodeName); ok {
				if pod.Status.Phase != corev1.PodRunning {
					return false, fmt.Errorf("want pod %s phase is running, got %s", pod.Name, pod.Status.Phase)
				}
				if pods.need(&list.Items[index]) {
					if pod.Status.HostIP != nodeInfo.HostIPs[0] {
						return false, fmt.Errorf("want pod %s hostIP=%s, got %s", pod.Name, nodeInfo.HostIPs[0], pod.Status.HostIP)
					}
					if pod.Spec.HostNetwork {
						if pod.Status.PodIP != nodeInfo.HostIPs[0] {
							return false, fmt.Errorf("want pod %s podIP=%s, got %s", pod.Name, nodeInfo.HostIPs[0], pod.Status.PodIP)
						}
					} else {
						cidr, _ := parseCIDR(nodeInfo.PodCIDRs[0])
						if !cidr.Contains(net.ParseIP(pod.Status.PodIP)) {
							return false, fmt.Errorf("want pod %s podIP=%s in %s, got not", pod.Name, pod.Status.PodIP, nodeInfo.PodCIDRs[0])
						}
					}
				}
			} else if pod.Status.Phase == corev1.PodRunning {
				return false, fmt.Errorf("want pod %s phase is not running, got %s", pod.Name, pod.Status.Phase)
			}
		}
		return true, nil
	}, wait.WithContinueOnError(5))
	if err != nil {
		t.Fatal(err)
	}

	pod, ok := slices.Find(list.Items, func(pod corev1.Pod) bool {
		return pod.Name == "pod0"
	})
	if !ok {
		t.Fatal(fmt.Errorf("not found pod0"))
	}
	now := metav1.Now()
	pod.DeletionTimestamp = &now
	_, err = clientset.CoreV1().Pods("default").Update(ctx, &pod, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("delete pod error: %w", err))
	}

	err = wait.Poll(ctx, func(ctx context.Context) (done bool, err error) {
		list, err = clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("list pods error: %w", err)
		}
		if len(list.Items) != 4 {
			return false, fmt.Errorf("want 4 pods, got %d", len(list.Items))
		}
		return true, nil
	}, wait.WithContinueOnError(10))
	if err != nil {
		t.Fatal(err)
	}
}
