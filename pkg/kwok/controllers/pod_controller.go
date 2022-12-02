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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/pager"

	"sigs.k8s.io/kwok/pkg/cni"
	"sigs.k8s.io/kwok/pkg/log"
)

var (
	removeFinalizers = []byte(`{"metadata":{"finalizers":null}}`)
	deleteOpt        = *metav1.NewDeleteOptions(0)
	podFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
)

// PodController is a fake pods implementation that can be used to test
type PodController struct {
	enableCNI                             bool
	clientSet                             kubernetes.Interface
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	nodeIP                                string
	cidrIPNet                             *net.IPNet
	nodeHasFunc                           func(nodeName string) bool
	ipPool                                *ipPool
	podStatusTemplate                     string
	renderer                              *renderer
	lockPodChan                           chan *corev1.Pod
	lockPodParallelism                    int
	deletePodChan                         chan *corev1.Pod
	deletePodParallelism                  int
}

// PodControllerConfig is the configuration for the PodController
type PodControllerConfig struct {
	EnableCNI                             bool
	ClientSet                             kubernetes.Interface
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	NodeIP                                string
	CIDR                                  string
	NodeHasFunc                           func(nodeName string) bool
	PodStatusTemplate                     string
	LockPodParallelism                    int
	DeletePodParallelism                  int
	FuncMap                               template.FuncMap
}

// NewPodController creates a new fake pods controller
func NewPodController(conf PodControllerConfig) (*PodController, error) {
	cidrIPNet, err := parseCIDR(conf.CIDR)
	if err != nil {
		return nil, err
	}

	disregardStatusWithAnnotationSelector, err := labelsParse(conf.DisregardStatusWithAnnotationSelector)
	if err != nil {
		return nil, err
	}

	disregardStatusWithLabelSelector, err := labelsParse(conf.DisregardStatusWithLabelSelector)
	if err != nil {
		return nil, err
	}

	n := &PodController{
		enableCNI:                             conf.EnableCNI,
		clientSet:                             conf.ClientSet,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		nodeIP:                                conf.NodeIP,
		cidrIPNet:                             cidrIPNet,
		ipPool:                                newIPPool(cidrIPNet),
		nodeHasFunc:                           conf.NodeHasFunc,
		podStatusTemplate:                     conf.PodStatusTemplate,
		lockPodChan:                           make(chan *corev1.Pod),
		lockPodParallelism:                    conf.LockPodParallelism,
		deletePodChan:                         make(chan *corev1.Pod),
		deletePodParallelism:                  conf.DeletePodParallelism,
	}
	funcMap = template.FuncMap{
		"NodeIP": func() string {
			return n.nodeIP
		},
		"PodIP": func() string {
			return n.ipPool.Get()
		},
	}
	for k, v := range conf.FuncMap {
		funcMap[k] = v
	}
	n.renderer = newRenderer(funcMap)
	return n, nil
}

// Start starts the fake pod controller
// It will modify the pods status to we want
func (c *PodController) Start(ctx context.Context) error {
	go c.LockPods(ctx, c.lockPodChan)
	go c.DeletePods(ctx, c.deletePodChan)

	opt := metav1.ListOptions{
		FieldSelector: podFieldSelector,
	}
	err := c.WatchPods(ctx, c.lockPodChan, c.deletePodChan, opt)
	if err != nil {
		return fmt.Errorf("failed watch pods: %w", err)
	}

	logger := log.FromContext(ctx)
	go func() {
		err = c.ListPods(ctx, c.lockPodChan, opt)
		if err != nil {
			logger.Error("Failed list pods", err)
		}
	}()
	return nil
}

// DeletePod deletes a pod
func (c *PodController) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	if len(pod.Finalizers) != 0 {
		_, err := c.clientSet.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.MergePatchType, removeFinalizers, metav1.PatchOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Warn("Patch pod", err)
				return nil
			}
			return err
		}
	}

	err := c.clientSet.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOpt)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Delete pod", err)
			return nil
		}
		return err
	}

	logger.Info("Delete pod")
	return nil
}

// DeletePods deletes pods from the channel
func (c *PodController) DeletePods(ctx context.Context, pods <-chan *corev1.Pod) {
	tasks := newParallelTasks(c.lockPodParallelism)
	logger := log.FromContext(ctx)
	for pod := range pods {
		localPod := pod
		tasks.Add(func() {
			err := c.DeletePod(ctx, localPod)
			if err != nil {
				logger.Error("Failed to delete pod", err,
					"pod", log.KObj(localPod),
					"node", localPod.Spec.NodeName,
				)
			}
		})
	}
	tasks.Wait()
}

// LockPod locks a given pod
func (c *PodController) LockPod(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	patch, err := c.configurePod(pod)
	if err != nil {
		return err
	}
	if patch == nil {
		logger.Info("Skip pod",
			"reason", "do not need to modify",
		)
		return nil
	}
	_, err = c.clientSet.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch pod", err)
			return nil
		}
		return err
	}
	logger.Info("Lock pod")
	return nil
}

// LockPods locks a pods from the channel
func (c *PodController) LockPods(ctx context.Context, pods <-chan *corev1.Pod) {
	logger := log.FromContext(ctx)
	tasks := newParallelTasks(c.lockPodParallelism)
	for pod := range pods {
		localPod := pod
		tasks.Add(func() {
			err := c.LockPod(ctx, localPod)
			if err != nil {
				logger.Error("Failed to lock pod", err,
					"pod", log.KObj(localPod),
					"node", localPod.Spec.NodeName,
				)
			}
		})
	}
	tasks.Wait()
}

func (c *PodController) needLockPod(pod *corev1.Pod) bool {
	if !c.nodeHasFunc(pod.Spec.NodeName) {
		return false
	}

	if c.disregardStatusWithAnnotationSelector != nil &&
		len(pod.Annotations) != 0 &&
		c.disregardStatusWithAnnotationSelector.Matches(labels.Set(pod.Annotations)) {
		return false
	}

	if c.disregardStatusWithLabelSelector != nil &&
		len(pod.Labels) != 0 &&
		c.disregardStatusWithLabelSelector.Matches(labels.Set(pod.Labels)) {
		return false
	}
	return true
}

// WatchPods watch pods put into the channel
func (c *PodController) WatchPods(ctx context.Context, lockChan, deleteChan chan<- *corev1.Pod, opt metav1.ListOptions) error {
	watcher, err := c.clientSet.CoreV1().Pods(corev1.NamespaceAll).Watch(ctx, opt)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	go func() {
		rc := watcher.ResultChan()
	loop:
		for {
			select {
			case event, ok := <-rc:
				if !ok {
					for {
						watcher, err := c.clientSet.CoreV1().Pods(corev1.NamespaceAll).Watch(ctx, opt)
						if err == nil {
							rc = watcher.ResultChan()
							continue loop
						}

						logger.Error("Failed to watch pods", err)
						select {
						case <-ctx.Done():
							break loop
						case <-time.After(time.Second * 5):
						}
					}
				}
				switch event.Type {
				case watch.Added:
					pod := event.Object.(*corev1.Pod)
					if c.needLockPod(pod) {
						lockChan <- pod.DeepCopy()
					} else {
						logger.Info("Skip pod",
							"reason", "not manage",
							"event", event.Type,
							"pod", log.KObj(pod),
							"node", pod.Spec.NodeName,
						)
					}
				case watch.Modified:
					pod := event.Object.(*corev1.Pod)

					// At a Kubelet, we need to delete this pod on the node we manage
					if pod.DeletionTimestamp != nil {
						if c.nodeHasFunc(pod.Spec.NodeName) {
							deleteChan <- pod.DeepCopy()
						} else {
							logger.Info("Skip pod",
								"reason", "not manage",
								"event", event.Type,
								"pod", log.KObj(pod),
								"node", pod.Spec.NodeName,
							)
						}
					} else {
						if c.needLockPod(pod) {
							lockChan <- pod.DeepCopy()
						} else {
							logger.Info("Skip pod",
								"reason", "not manage",
								"event", event.Type,
								"pod", log.KObj(pod),
								"node", pod.Spec.NodeName,
							)
						}
					}
				case watch.Deleted:
					pod := event.Object.(*corev1.Pod)
					if c.nodeHasFunc(pod.Spec.NodeName) {
						if !c.enableCNI {
							// Recycling PodIP
							if pod.Status.PodIP != "" && c.cidrIPNet.Contains(net.ParseIP(pod.Status.PodIP)) {
								c.ipPool.Put(pod.Status.PodIP)
							}
						} else {
							err := cni.Remove(context.Background(), string(pod.UID), pod.Name, pod.Namespace)
							if err != nil {
								logger.Error("cni remove", err)
							}
						}
					}
				}
			case <-ctx.Done():
				watcher.Stop()
				break loop
			}
		}
		logger.Info("Stop watch pods")
	}()

	return nil
}

// ListPods list pods put into the channel
func (c *PodController) ListPods(ctx context.Context, ch chan<- *corev1.Pod, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientSet.CoreV1().Pods(corev1.NamespaceAll).List(ctx, opts)
	})
	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		pod := obj.(*corev1.Pod)
		if c.needLockPod(pod) {
			ch <- pod.DeepCopy()
		}
		return nil
	})
}

// LockPodsOnNode locks pods on the node
func (c *PodController) LockPodsOnNode(ctx context.Context, nodeName string) error {
	return c.ListPods(ctx, c.lockPodChan, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
	})
}

func (c *PodController) configurePod(pod *corev1.Pod) ([]byte, error) {
	if !c.enableCNI {
		// Mark the pod IP that existed before the kubelet was started
		if c.cidrIPNet.Contains(net.ParseIP(pod.Status.PodIP)) {
			c.ipPool.Use(pod.Status.PodIP)
		}
	} else if pod.Status.PodIP == "" {
		ips, err := cni.Setup(context.Background(), string(pod.UID), pod.Name, pod.Namespace)
		if err != nil {
			return nil, err
		}
		pod.Status.PodIP = ips[0]
	}

	patch, err := c.computePatchData(pod, c.podStatusTemplate)
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return nil, nil
	}

	return json.Marshal(map[string]json.RawMessage{
		"status": patch,
	})
}

func (c *PodController) computePatchData(pod *corev1.Pod, temp string) ([]byte, error) {
	patch, err := c.renderer.renderToJson(temp, pod)
	if err != nil {
		return nil, err
	}

	// Check whether the pod need to be patch
	if pod.Status.Phase != corev1.PodPending {
		original, err := json.Marshal(pod.Status)
		if err != nil {
			return nil, err
		}

		sum, err := strategicpatch.StrategicMergePatch(original, patch, pod.Status)
		if err != nil {
			return nil, err
		}

		podStatus := corev1.PodStatus{}
		err = json.Unmarshal(sum, &podStatus)
		if err != nil {
			return nil, err
		}

		dist, err := json.Marshal(podStatus)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(original, dist) {
			return nil, nil
		}
	}

	return patch, nil
}
