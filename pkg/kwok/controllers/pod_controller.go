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
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/tasks"
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
	parallelPriority                      int
	delayParallelPriority                 int
	nodeIP                                string
	defaultCIDR                           string
	nodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	ipPools                               maps.SyncMap[string, *ipPool]
	renderer                              *renderer
	lockPodChan                           chan *corev1.Pod
	tasks                                 *tasks.ParallelPriorityTasks
	lifecycle                             Lifecycle
	cronjob                               *cron.Cron
	delayJobsCancels                      maps.SyncMap[string, cron.DoFunc]
	recorder                              record.EventRecorder
}

// PodControllerConfig is the configuration for the PodController
type PodControllerConfig struct {
	EnableCNI                             bool
	ClientSet                             kubernetes.Interface
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	ParallelPriority                      int
	DelayParallelPriority                 int
	NodeIP                                string
	CIDR                                  string
	NodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	Stages                                []*internalversion.Stage
	Tasks                                 *tasks.ParallelPriorityTasks
	FuncMap                               template.FuncMap
	Recorder                              record.EventRecorder
}

// NewPodController creates a new fake pods controller
func NewPodController(conf PodControllerConfig) (*PodController, error) {
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

	c := &PodController{
		enableCNI:                             conf.EnableCNI,
		clientSet:                             conf.ClientSet,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		parallelPriority:                      conf.ParallelPriority,
		delayParallelPriority:                 conf.DelayParallelPriority,
		nodeIP:                                conf.NodeIP,
		defaultCIDR:                           conf.CIDR,
		nodeGetFunc:                           conf.NodeGetFunc,
		cronjob:                               cron.NewCron(),
		lifecycle:                             lifecycles,
		tasks:                                 conf.Tasks,
		lockPodChan:                           make(chan *corev1.Pod),
		recorder:                              conf.Recorder,
	}
	funcMap := template.FuncMap{
		"NodeIP":     c.funcNodeIP,
		"PodIP":      c.funcPodIP,
		"NodeIPWith": c.funcNodeIPWith,
		"PodIPWith":  c.funcPodIPWith,
	}
	for k, v := range conf.FuncMap {
		funcMap[k] = v
	}
	c.renderer = newRenderer(funcMap)
	return c, nil
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
		logger.Debug("Skip pod",
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
				logger.Debug("Skip pod",
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

	logger.Debug("Delayed play stage",
		"delay", delay,
		"stage", stageName,
	)
	cancelFunc, ok := c.cronjob.AddWithCancel(cron.Order(time.Now().Add(delay)), func() {
		c.tasks.Add(c.delayParallelPriority, func() {
			cancelOld, ok := c.delayJobsCancels.LoadAndDelete(key)
			if ok {
				cancelOld()
			}
			do()
		})
	})
	if ok {
		cancelOld, ok := c.delayJobsCancels.LoadOrStore(key, cancelFunc)
		if ok {
			cancelOld()
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
		c.tasks.Add(c.parallelPriority, func() {
			err := c.LockPod(ctx, localPod)
			if err != nil {
				logger.Error("Failed to lock pod", err,
					"pod", log.KObj(localPod),
					"node", localPod.Spec.NodeName,
				)
			}
		})
	}
}

func (c *PodController) needLockPod(pod *corev1.Pod) bool {
	if _, has := c.nodeGetFunc(pod.Spec.NodeName); !has {
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
						logger.Debug("Skip pod",
							"reason", "not managed",
							"event", event.Type,
							"pod", log.KObj(pod),
							"node", pod.Spec.NodeName,
						)
					}

				case watch.Deleted:
					pod := event.Object.(*corev1.Pod)
					// Recycling PodIP
					c.recyclingPodIP(ctx, pod)
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

// ipPool returns the ipPool for the given cidr
func (c *PodController) ipPool(cidr string) (*ipPool, error) {
	pool, ok := c.ipPools.Load(cidr)
	if ok {
		return pool, nil
	}
	ipnet, err := parseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	pool = newIPPool(ipnet)
	c.ipPools.Store(cidr, pool)
	return pool, nil
}

// recyclingPodIP recycling pod ip
func (c *PodController) recyclingPodIP(ctx context.Context, pod *corev1.Pod) {
	// Skip host network
	if pod.Spec.HostNetwork {
		return
	}

	// Skip not managed node
	nodeInfo, ok := c.nodeGetFunc(pod.Spec.NodeName)
	if !ok {
		return
	}

	logger := log.FromContext(ctx)
	if !c.enableCNI {
		if pod.Status.PodIP != "" {
			cidr := c.defaultCIDR
			if len(nodeInfo.PodCIDRs) > 0 {
				cidr = nodeInfo.PodCIDRs[0]
			}
			pool, err := c.ipPool(cidr)
			if err != nil {
				logger.Error("Failed to get ip pool", err,
					"pod", log.KObj(pod),
					"node", pod.Spec.NodeName,
				)
			} else {
				pool.Put(pod.Status.PodIP)
			}
		}
	} else {
		err := cni.Remove(context.Background(), string(pod.UID), pod.Name, pod.Namespace)
		if err != nil {
			logger.Error("cni remove", err)
		}
	}
}

func (c *PodController) configurePod(pod *corev1.Pod, template string) ([]byte, error) {
	if !c.enableCNI {
		// Mark the pod IP that existed before the kubelet was started
		if nodeInfo, has := c.nodeGetFunc(pod.Spec.NodeName); has {
			if pod.Status.PodIP != "" {
				cidr := c.defaultCIDR
				if len(nodeInfo.PodCIDRs) > 0 {
					cidr = nodeInfo.PodCIDRs[0]
				}
				pool, err := c.ipPool(cidr)
				if err == nil {
					pool.Use(pod.Status.PodIP)
				}
			}
		}
	}

	patch, err := c.computePatch(pod, template)
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

func (c *PodController) computePatch(pod *corev1.Pod, tpl string) ([]byte, error) {
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

func (c *PodController) funcNodeIP() string {
	return c.nodeIP
}

func (c *PodController) funcNodeIPWith(nodeName string) string {
	nodeInfo, has := c.nodeGetFunc(nodeName)
	if has && len(nodeInfo.HostIPs) > 0 {
		hostIP := nodeInfo.HostIPs[0]
		if hostIP != "" {
			return hostIP
		}
	}
	return c.nodeIP
}

func (c *PodController) funcPodIP() string {
	podCIDR := c.defaultCIDR
	pool, err := c.ipPool(podCIDR)
	if err == nil {
		return pool.Get()
	}
	return c.nodeIP
}

func (c *PodController) funcPodIPWith(nodeName string, hostNetwork bool, uid, name, namespace string) (string, error) {
	if hostNetwork {
		return c.funcNodeIPWith(nodeName), nil
	}

	if c.enableCNI {
		ips, err := cni.Setup(context.Background(), uid, name, namespace)
		if err != nil {
			return "", err
		}
		return ips[0], nil
	}

	podCIDR := c.defaultCIDR
	nodeInfo, has := c.nodeGetFunc(nodeName)
	if has && len(nodeInfo.PodCIDRs) > 0 {
		podCIDR = nodeInfo.PodCIDRs[0]
	}

	pool, err := c.ipPool(podCIDR)
	if err == nil {
		return pool.Get(), nil
	}
	return c.nodeIP, nil
}
