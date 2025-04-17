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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	podfast "sigs.k8s.io/kwok/kustomize/stage/pod/fast"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
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

	wantHostIPFunc := func(node corev1.Node) string {
		nodeIP := defaultNodeIP
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodeIP = addr.Address
				break
			}
		}
		return nodeIP
	}

	wantPodCIDRFunc := func(node corev1.Node) string {
		podCIDR := defaultPodCIDR
		if node.Spec.PodCIDR != "" {
			podCIDR = node.Spec.PodCIDR
		}
		return podCIDR
	}
	nodeGetFunc := func(nodeName string) (*NodeInfo, bool) {
		_, err := clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, false
		}

		nodeInfo := &NodeInfo{}
		return nodeInfo, true
	}

	podStages, _ := slices.MapWithError([]string{
		podfast.DefaultPodReady,
		podfast.DefaultPodComplete,
		podfast.DefaultPodDelete,
	}, config.UnmarshalWithType[*internalversion.Stage, string])

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
	nodeCache, err := nodesInformer.WatchWithCache(ctx, informer.Option{}, nodeCh)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to watch nodes: %w", err))
	}

	lc, _ := lifecycle.NewLifecycle(podStages)
	annotationSelector, _ := labels.Parse("fake=custom")
	pods, err := NewPodController(PodControllerConfig{
		TypedClient:                           clientset,
		NodeCacheGetter:                       nodeCache,
		NodeIP:                                defaultNodeIP,
		CIDR:                                  defaultPodCIDR,
		DisregardStatusWithAnnotationSelector: annotationSelector.String(),
		Lifecycle:                             resources.NewStaticGetter(lc),
		NodeGetFunc:                           nodeGetFunc,
		PlayStageParallelism:                  2,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new pods controller error: %w", err))
	}

	podsCh := make(chan informer.Event[*corev1.Pod], 1)
	podsCli := clientset.CoreV1().Pods(corev1.NamespaceAll)
	podsInformer := informer.NewInformer[*corev1.Pod, *corev1.PodList](podsCli)
	err = podsInformer.Watch(ctx, informer.Option{
		FieldSelector: fields.OneTermNotEqualSelector("spec.nodeName", "").String(),
	}, podsCh)
	if err != nil {
		t.Fatal(fmt.Errorf("watch pods error: %w", err))
	}

	err = pods.Start(ctx, podsCh)
	if err != nil {
		t.Fatal(fmt.Errorf("start pods controller error: %w", err))
	}

	listNodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("list nodes error: %w", err))
	}
	for _, node := range listNodes.Items {
		if _, ok := nodeGetFunc(node.Name); ok {
			wantPodCIRD := wantPodCIDRFunc(node)
			wantHostIP := wantHostIPFunc(node)

			if node.Spec.PodCIDR != "" {
				if node.Spec.PodCIDR != wantPodCIRD {
					t.Fatal(fmt.Errorf("want node %s podCIDR=%s, got %s", node.Name, node.Spec.PodCIDR, wantPodCIRD))
				}
			} else {
				if defaultPodCIDR != wantPodCIRD {
					t.Fatal(fmt.Errorf("want node %s podCIDR=%s, got %s", node.Name, defaultPodCIDR, wantPodCIRD))
				}
			}

			if len(node.Status.Addresses) != 0 {
				if node.Status.Addresses[0].Address != wantHostIP {
					t.Fatal(fmt.Errorf("want node %s address=%s, got %s", node.Name, node.Status.Addresses[0].Address, wantHostIP))
				}
			} else {
				if defaultNodeIP != wantHostIP {
					t.Fatal(fmt.Errorf("want node %s address=%s, got %s", node.Name, defaultNodeIP, wantHostIP))
				}
			}
		}
	}

	time.Sleep(1 * time.Second)
	_, err = clientset.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pod1",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "1",
					Image: "test-image",
				},
			},
			NodeName: "node0",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(fmt.Errorf("create pod1 error: %w", err))
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
			if _, ok := nodeGetFunc(pod.Spec.NodeName); ok {
				if pod.Status.Phase != corev1.PodRunning {
					return false, fmt.Errorf("want pod %s phase is running, got %s", pod.Name, pod.Status.Phase)
				}

				node, err := clientset.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
				if err != nil {
					return false, fmt.Errorf("get node %s error: %w", pod.Spec.NodeName, err)
				}

				if pods.need(&list.Items[index]) {
					wantHostIP := wantHostIPFunc(*node)
					wantPodCIRD := wantPodCIDRFunc(*node)
/*add below some code for retry checking of hostIP*/
					retryCount := 0
					maxRetries := 5
					for retryCount < maxRetries {
						if pod.Status.HostIP != wantHostIP {
							if retryCount == maxRetries-1 {
								return false, fmt.Errorf("want pod %s hostIP=%s, got %s", pod.Name, wantHostIP, pod.Status.HostIP)
							}
							retryCount++
							time.Sleep(100 * time.Millisecond)
							continue
						}
						break
					}
/*add above some code for retry checking of hostIP*/					
					if pod.Status.HostIP != wantHostIP {
						return false, fmt.Errorf("want pod %s hostIP=%s, got %s", pod.Name, wantHostIP, pod.Status.HostIP)
					}
					if pod.Spec.HostNetwork {
						if pod.Status.PodIP != wantHostIP {
							return false, fmt.Errorf("want pod %s podIP=%s, got %s", pod.Name, wantHostIP, pod.Status.PodIP)
						}
					} else {
						cidr, _ := parseCIDR(wantPodCIRD)
						if !cidr.Contains(net.ParseIP(pod.Status.PodIP)) {
							return false, fmt.Errorf("want pod %s podIP=%s in %s, got not", pod.Name, pod.Status.PodIP, wantPodCIRD)
						}
					}
				}
			} else if pod.Status.Phase == corev1.PodRunning {
				return false, fmt.Errorf("want pod %s phase is not running, got %s", pod.Name, pod.Status.Phase)
			}
		}
		return true, nil
	}, wait.WithTimeout(5*time.Second)) //update this functionality
	if err != nil {
		t.Fatal(err)
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
