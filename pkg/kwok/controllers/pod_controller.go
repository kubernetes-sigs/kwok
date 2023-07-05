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
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/kwok/cni"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/maps"
)

var (
	deleteOpt        = *metav1.NewDeleteOptions(0)
	podFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
)

// PodController is a fake pods implementation that can be used to test
type PodController struct {
	clock                                 clock.Clock
	enableCNI                             bool
	typedClient                           kubernetes.Interface
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	nodeIP                                string
	defaultCIDR                           string
	namespace                             string
	nodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	nodeHasMetric                         func(nodeName string) bool
	ipPools                               maps.SyncMap[string, *ipPool]
	renderer                              gotpl.Renderer
	preprocessChan                        chan *corev1.Pod
	playStageChan                         chan resourceStageJob[*corev1.Pod]
	playStageParallelism                  uint
	lifecycle                             resources.Getter[Lifecycle]
	cronjob                               *cron.Cron
	delayJobs                             jobInfoMap
	recorder                              record.EventRecorder
	readOnlyFunc                          func(nodeName string) bool
	triggerPreprocessChan                 chan string
}

// PodControllerConfig is the configuration for the PodController
type PodControllerConfig struct {
	Clock                                 clock.Clock
	EnableCNI                             bool
	TypedClient                           kubernetes.Interface
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	NodeIP                                string
	CIDR                                  string
	Namespace                             string
	NodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	NodeHasMetric                         func(nodeName string) bool
	Lifecycle                             resources.Getter[Lifecycle]
	PlayStageParallelism                  uint
	FuncMap                               gotpl.FuncMap
	Recorder                              record.EventRecorder
	ReadOnlyFunc                          func(nodeName string) bool
}

// NewPodController creates a new fake pods controller
func NewPodController(conf PodControllerConfig) (*PodController, error) {
	if conf.PlayStageParallelism <= 0 {
		return nil, fmt.Errorf("playStageParallelism must be greater than 0")
	}

	disregardStatusWithAnnotationSelector, err := labelsParse(conf.DisregardStatusWithAnnotationSelector)
	if err != nil {
		return nil, err
	}

	disregardStatusWithLabelSelector, err := labelsParse(conf.DisregardStatusWithLabelSelector)
	if err != nil {
		return nil, err
	}

	if conf.Clock == nil {
		conf.Clock = clock.RealClock{}
	}

	c := &PodController{
		clock:                                 conf.Clock,
		enableCNI:                             conf.EnableCNI,
		typedClient:                           conf.TypedClient,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		nodeIP:                                conf.NodeIP,
		defaultCIDR:                           conf.CIDR,
		namespace:                             conf.Namespace,
		nodeGetFunc:                           conf.NodeGetFunc,
		cronjob:                               cron.NewCron(),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *corev1.Pod),
		triggerPreprocessChan:                 make(chan string, 16),
		playStageChan:                         make(chan resourceStageJob[*corev1.Pod]),
		recorder:                              conf.Recorder,
		readOnlyFunc:                          conf.ReadOnlyFunc,
	}
	funcMap := gotpl.FuncMap{
		"NodeIP":     c.funcNodeIP,
		"PodIP":      c.funcPodIP,
		"NodeIPWith": c.funcNodeIPWith,
		"PodIPWith":  c.funcPodIPWith,
	}
	for k, v := range conf.FuncMap {
		funcMap[k] = v
	}
	c.renderer = gotpl.NewRenderer(funcMap)
	return c, nil
}

// Start starts the fake pod controller
// It will modify the pods status to we want
func (c *PodController) Start(ctx context.Context) error {
	go c.preprocessWorker(ctx)
	go c.triggerPreprocessWorker(ctx)
	for i := uint(0); i < c.playStageParallelism; i++ {
		go c.playStageWorker(ctx)
	}

	opt := metav1.ListOptions{
		FieldSelector: podFieldSelector,
	}
	err := c.watchResources(ctx, opt)
	if err != nil {
		return fmt.Errorf("failed watch pods: %w", err)
	}

	logger := log.FromContext(ctx)
	go func() {
		err = c.listResources(ctx, opt)
		if err != nil {
			logger.Error("Failed list pods", err)
		}
	}()
	return nil
}

// finalizersModify modify the finalizers of the pod
func (c *PodController) finalizersModify(ctx context.Context, pod *corev1.Pod, finalizers *internalversion.StageFinalizers) (*corev1.Pod, error) {
	ops := finalizersModify(pod.Finalizers, finalizers)
	if len(ops) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	result, err := c.typedClient.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch pod finalizers",
				"err", err,
			)
			return nil, nil
		}
		return nil, err
	}
	logger.Info("Patch pod finalizers")
	return result, nil
}

// deleteResource deletes a pod
func (c *PodController) deleteResource(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	err := c.typedClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOpt)
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

// preprocessWorker receives the resource from the preprocessChan and preprocess it
func (c *PodController) preprocessWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop preprocess worker")
			return
		case pod := <-c.preprocessChan:
			err := c.preprocess(ctx, pod)
			if err != nil {
				logger.Error("Failed to preprocess node", err,
					"pod", log.KObj(pod),
					"node", pod.Spec.NodeName,
				)
			}
		}
	}
}

// triggerPreprocessWorker receives the resource from the triggerPreprocessChan and preprocess it
func (c *PodController) triggerPreprocessWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop trigger preprocess worker")
			return
		case nodeName := <-c.triggerPreprocessChan:
			err := c.listResources(ctx, metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
			})
			if err != nil {
				logger.Error("Failed to preprocess node", err,
					"node", nodeName,
				)
			}
		}
	}
}

// preprocess the pod and send it to the playStageWorker
func (c *PodController) preprocess(ctx context.Context, pod *corev1.Pod) error {
	key := log.KObj(pod).String()

	resourceJob, ok := c.delayJobs.Load(key)
	if ok && resourceJob.ResourceVersion == pod.ResourceVersion {
		return nil
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", key,
		"node", pod.Spec.NodeName,
	)

	data, err := expression.ToJSONStandard(pod)
	if err != nil {
		return err
	}

	lifecycle := c.lifecycle.Get()
	stage, err := lifecycle.Match(pod.Labels, pod.Annotations, data)
	if err != nil {
		return fmt.Errorf("stage match: %w", err)
	}
	if stage == nil {
		logger.Debug("Skip pod",
			"reason", "not match any stages",
		)
		return nil
	}

	now := c.clock.Now()
	delay, _ := stage.Delay(ctx, data, now)

	if delay != 0 {
		stageName := stage.Name()
		logger.Debug("Delayed play stage",
			"delay", delay,
			"stage", stageName,
		)
	}

	cancelFunc, ok := c.cronjob.AddWithCancel(cron.Order(now.Add(delay)), func() {
		resourceJob, ok := c.delayJobs.LoadAndDelete(key)
		if ok {
			resourceJob.Cancel()
		}
		c.playStageChan <- resourceStageJob[*corev1.Pod]{
			Resource: pod,
			Stage:    stage,
		}
	})
	if ok {
		resourceJob, ok := c.delayJobs.LoadOrStore(key, jobInfo{
			ResourceVersion: pod.ResourceVersion,
			Cancel:          cancelFunc,
		})
		if ok {
			resourceJob.Cancel()
		}
	}

	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *PodController) playStageWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop play stage worker")
			return
		case pod := <-c.playStageChan:
			c.playStage(ctx, pod.Resource, pod.Stage)
		}
	}
}

// playStage plays the stage
func (c *PodController) playStage(ctx context.Context, pod *corev1.Pod, stage *LifecycleStage) {
	next := stage.Next()
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
		"stage", stage.Name(),
	)

	if next.Event != nil && c.recorder != nil {
		c.recorder.Event(&corev1.ObjectReference{
			Kind:      "Pod",
			UID:       pod.UID,
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}, next.Event.Type, next.Event.Reason, next.Event.Message)
	}
	if next.Finalizers != nil {
		result, err := c.finalizersModify(ctx, pod, next.Finalizers)
		if err != nil {
			logger.Error("Failed to finalizers", err)
		}
		if result != nil && stage.ImmediateNextStage() {
			c.preprocessChan <- result
		}
	}
	if next.Delete {
		err := c.deleteResource(ctx, pod)
		if err != nil {
			logger.Error("Failed to delete pod", err)
		}
	} else if next.StatusTemplate != "" {
		patch, err := c.configureResource(pod, next.StatusTemplate)
		if err != nil {
			logger.Error("Failed to configure pod", err)
			return
		}
		if patch == nil {
			logger.Debug("Skip pod",
				"reason", "do not need to modify",
			)
		} else {
			result, err := c.patchResource(ctx, pod, patch)
			if err != nil {
				logger.Error("Failed to patch node", err)
			}
			if result != nil && stage.ImmediateNextStage() {
				c.preprocessChan <- result
			}
		}
	}
}

func (c *PodController) readOnly(nodeName string) bool {
	if c.readOnlyFunc == nil {
		return false
	}
	return c.readOnlyFunc(nodeName)
}

// patchResource patches the resource
func (c *PodController) patchResource(ctx context.Context, pod *corev1.Pod, patch []byte) (*corev1.Pod, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	result, err := c.typedClient.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch pod",
				"err", err,
			)
			return nil, nil
		}
		return nil, err
	}
	logger.Info("Patch pod")
	return result, nil
}

func (c *PodController) need(pod *corev1.Pod) bool {
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

// watchResources watch resources and send to preprocessChan
func (c *PodController) watchResources(ctx context.Context, opt metav1.ListOptions) error {
	watcher, err := c.typedClient.CoreV1().Pods(c.namespace).Watch(ctx, opt)
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
						watcher, err := c.typedClient.CoreV1().Pods(c.namespace).Watch(ctx, opt)
						if err == nil {
							rc = watcher.ResultChan()
							continue loop
						}

						logger.Error("Failed to watch pods", err)
						select {
						case <-ctx.Done():
							break loop
						case <-c.clock.After(time.Second * 5):
						}
					}
				}

				switch event.Type {
				case watch.Added, watch.Modified:
					pod := event.Object.(*corev1.Pod)
					if c.need(pod) {
						if c.readOnly(pod.Spec.NodeName) {
							logger.Debug("Skip pod",
								"reason", "read only",
								"event", event.Type,
								"pod", log.KObj(pod),
								"node", pod.Spec.NodeName,
							)
						} else {
							c.preprocessChan <- pod.DeepCopy()
						}
					} else {
						logger.Debug("Skip pod",
							"reason", "not managed",
							"event", event.Type,
							"pod", log.KObj(pod),
							"node", pod.Spec.NodeName,
						)
					}

					if event.Type == watch.Added &&
						c.nodeHasMetric != nil &&
						c.nodeHasMetric(pod.Spec.NodeName) &&
						c.nodeGetFunc != nil {
						nodeInfo, ok := c.nodeGetFunc(pod.Spec.NodeName)
						if ok {
							nodeInfo.StartedContainer.Add(int64(len(pod.Spec.Containers)))
						}
					}
				case watch.Deleted:
					pod := event.Object.(*corev1.Pod)
					if c.need(pod) {
						// Recycling PodIP
						c.recyclingPodIP(ctx, pod)

						// Cancel delay job
						key := log.KObj(pod).String()
						resourceJob, ok := c.delayJobs.LoadAndDelete(key)
						if ok {
							resourceJob.Cancel()
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

// listResources lists all resources and sends to preprocessChan
func (c *PodController) listResources(ctx context.Context, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.typedClient.CoreV1().Pods(c.namespace).List(ctx, opts)
	})

	logger := log.FromContext(ctx)

	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		pod := obj.(*corev1.Pod)
		if c.need(pod) {
			if c.readOnly(pod.Spec.NodeName) {
				logger.Debug("Skip pod",
					"pod", log.KObj(pod),
					"node", pod.Spec.NodeName,
					"reason", "read only",
				)
			} else {
				c.preprocessChan <- pod.DeepCopy()
			}
		}
		return nil
	})
}

// PlayStagePodsOnNode plays stage pods on node
func (c *PodController) PlayStagePodsOnNode(nodeName string) {
	c.triggerPreprocessChan <- nodeName
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

func (c *PodController) configureResource(pod *corev1.Pod, template string) ([]byte, error) {
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
	patch, err := c.renderer.ToJSON(tpl, pod)
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
