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
	"sync"
	"text/template"
	"time"

	"github.com/wzshiming/cron"
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
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/cni"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
)

var (
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
	nodeGetFunc                           func(nodeName string) (NodeInfo, bool)
	ipPool                                *ipPool
	renderer                              *renderer
	lockPodChan                           chan *corev1.Pod
	parallelTasks                         *parallelTasks
	lifecycle                             Lifecycle
	cronjob                               *cron.Cron
	delayJobsCancels                      sync.Map
	recorder                              record.EventRecorder
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
	NodeGetFunc                           func(nodeName string) (NodeInfo, bool)
	Stages                                []*internalversion.Stage
	LockPodParallelism                    int
	FuncMap                               template.FuncMap
	Recorder                              record.EventRecorder
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

	lifecycles, err := NewLifecycle(conf.Stages)
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
		nodeGetFunc:                           conf.NodeGetFunc,
		cronjob:                               cron.NewCron(),
		lifecycle:                             lifecycles,
		parallelTasks:                         newParallelTasks(conf.LockPodParallelism),
		lockPodChan:                           make(chan *corev1.Pod),
		recorder:                              conf.Recorder,
	}
	funcMap := template.FuncMap{
		"NodeIP": func(nodeName string) string {
			nodeInfo, has := n.nodeGetFunc(nodeName)
			if has && len(nodeInfo.HostIPs) > 0 {
				hostIP := nodeInfo.HostIPs[0]
				if hostIP != "" {
					return hostIP
				}
			}
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

	opt := metav1.ListOptions{
		FieldSelector: podFieldSelector,
	}
	err := c.WatchPods(ctx, c.lockPodChan, opt)
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

// FinalizersModify modify the finalizers of the pod
func (c *PodController) FinalizersModify(ctx context.Context, pod *corev1.Pod, finalizers *internalversion.StageFinalizers) error {
	ops := finalizersModify(pod.Finalizers, finalizers)
	if len(ops) == 0 {
		return nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	_, err = c.clientSet.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch pod finalizers",
				"err", err,
			)
			return nil
		}
		return err
	}
	logger.Info("Patch pod finalizers")
	return nil
}

// DeletePod deletes a pod
func (c *PodController) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	err := c.clientSet.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOpt)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Delete pod",
				"err", err,
			)
			return nil
		}
		return err
	}

	logger.Info("Delete pod")
	return nil
}

// LockPod locks a given pod
func (c *PodController) LockPod(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	data, err := expression.ToJSONStandard(pod)
	if err != nil {
		return err
	}
	key := pod.Name + "." + pod.Namespace

	_, ok := c.delayJobsCancels.Load(key)
	if ok {
		return nil
	}

	stage, err := c.lifecycle.Match(pod.Labels, pod.Annotations, data)
	if err != nil {
		return fmt.Errorf("stage match: %w", err)
	}
	if stage == nil {
		logger.Info("Skip pod",
			"reason", "not match any stages",
		)
		return nil
	}
	now := time.Now()
	delay, _ := stage.Delay(ctx, data, now)
	stageName := stage.Name()
	next := stage.Next()

	do := func() {
		if next.Event != nil && c.recorder != nil {
			c.recorder.Event(&corev1.ObjectReference{
				Kind:      "Pod",
				UID:       pod.UID,
				Name:      pod.Name,
				Namespace: pod.Namespace,
			}, next.Event.Type, next.Event.Reason, next.Event.Message)
		}
		if next.Finalizers != nil {
			err = c.FinalizersModify(ctx, pod, next.Finalizers)
			if err != nil {
				logger.Error("Failed to finalizers", err)
			}
		}
		if next.Delete {
			err = c.DeletePod(ctx, pod)
			if err != nil {
				logger.Error("Failed to delete pod", err)
			}
		} else if next.StatusTemplate != "" {
			patch, err := c.configurePod(pod, next.StatusTemplate)
			if err != nil {
				logger.Error("Failed to configure pod", err)
				return
			}
			if patch == nil {
				logger.Info("Skip pod",
					"reason", "do not need to modify",
				)
			} else {
				err = c.lockPod(ctx, pod, patch)
				if err != nil {
					logger.Error("Failed to lock pod", err)
				}
			}
		}
	}

	if delay == 0 {
		do()
		return nil
	}

	logger.Info("Delayed play stage",
		"delay", delay,
		"stage", stageName,
	)
	cancelFunc, ok := c.cronjob.AddWithCancel(cron.Order(time.Now().Add(delay)), func() {
		c.parallelTasks.Add(func() {
			old, ok := c.delayJobsCancels.LoadAndDelete(key)
			if ok {
				old.(cron.DoFunc)()
			}
			do()
		})
	})
	if ok {
		old, ok := c.delayJobsCancels.LoadOrStore(key, cancelFunc)
		if ok {
			old.(cron.DoFunc)()
		}
	}

	return nil
}

func (c *PodController) lockPod(ctx context.Context, pod *corev1.Pod, patch []byte) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)
	_, err := c.clientSet.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch pod",
				"err", err,
			)
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
	for pod := range pods {
		localPod := pod
		c.parallelTasks.Add(func() {
			err := c.LockPod(ctx, localPod)
			if err != nil {
				logger.Error("Failed to lock pod", err,
					"pod", log.KObj(localPod),
					"node", localPod.Spec.NodeName,
				)
			}
		})
	}
	c.parallelTasks.Wait()
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
func (c *PodController) WatchPods(ctx context.Context, lockChan chan<- *corev1.Pod, opt metav1.ListOptions) error {
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
				case watch.Added, watch.Modified:
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

				case watch.Deleted:
					pod := event.Object.(*corev1.Pod)
					if c.nodeHasFunc(pod.Spec.NodeName) && !pod.Spec.HostNetwork {
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

func (c *PodController) configurePod(pod *corev1.Pod, template string) ([]byte, error) {
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

	patch, err := c.computePatchData(pod, template)
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

func (c *PodController) computePatchData(pod *corev1.Pod, tpl string) ([]byte, error) {
	patch, err := c.renderer.renderToJSON(tpl, pod)
	if err != nil {
		return nil, err
	}

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

	return patch, nil
}
