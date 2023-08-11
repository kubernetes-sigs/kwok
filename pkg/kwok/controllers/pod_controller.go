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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/kwok/cni"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
)

var (
	deleteOpt = *metav1.NewDeleteOptions(0)
)

// PodController is a fake pods implementation that can be used to test
type PodController struct {
	clock                                 clock.Clock
	enableCNI                             bool
	typedClient                           kubernetes.Interface
	nodeCacheGetter                       informer.Getter[*corev1.Node]
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	nodeIP                                string
	defaultCIDR                           string
	nodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	ipPools                               maps.SyncMap[string, *ipPool]
	renderer                              gotpl.Renderer
	podsSets                              maps.SyncMap[log.ObjectRef, *PodInfo]
	podsOnNode                            maps.SyncMap[string, *maps.SyncMap[log.ObjectRef, *PodInfo]]
	preprocessChan                        chan *corev1.Pod
	playStageParallelism                  uint
	lifecycle                             resources.Getter[Lifecycle]
	delayQueue                            queue.DelayingQueue[resourceStageJob[*corev1.Pod]]
	delayQueueMapping                     maps.SyncMap[string, resourceStageJob[*corev1.Pod]]
	recorder                              record.EventRecorder
	readOnlyFunc                          func(nodeName string) bool
	enableMetrics                         bool
}

// PodInfo is the collection of necessary pod information
type PodInfo struct {
}

// PodControllerConfig is the configuration for the PodController
type PodControllerConfig struct {
	Clock                                 clock.Clock
	EnableCNI                             bool
	TypedClient                           kubernetes.Interface
	NodeCacheGetter                       informer.Getter[*corev1.Node]
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	NodeIP                                string
	CIDR                                  string
	NodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	NodeHasMetric                         func(nodeName string) bool
	Lifecycle                             resources.Getter[Lifecycle]
	PlayStageParallelism                  uint
	FuncMap                               gotpl.FuncMap
	Recorder                              record.EventRecorder
	ReadOnlyFunc                          func(nodeName string) bool
	EnableMetrics                         bool
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
		nodeCacheGetter:                       conf.NodeCacheGetter,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		nodeIP:                                conf.NodeIP,
		defaultCIDR:                           conf.CIDR,
		nodeGetFunc:                           conf.NodeGetFunc,
		delayQueue:                            queue.NewDelayingQueue[resourceStageJob[*corev1.Pod]](conf.Clock),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *corev1.Pod),
		recorder:                              conf.Recorder,
		readOnlyFunc:                          conf.ReadOnlyFunc,
		enableMetrics:                         conf.EnableMetrics,
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
func (c *PodController) Start(ctx context.Context, events <-chan informer.Event[*corev1.Pod]) error {
	go c.preprocessWorker(ctx)
	for i := uint(0); i < c.playStageParallelism; i++ {
		go c.playStageWorker(ctx)
	}
	go c.watchResources(ctx, events)
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

// preprocess the pod and send it to the playStageWorker
func (c *PodController) preprocess(ctx context.Context, pod *corev1.Pod) error {
	key := log.KObj(pod).String()

	resourceJob, ok := c.delayQueueMapping.Load(key)
	if ok && resourceJob.Resource.ResourceVersion == pod.ResourceVersion {
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

	item := resourceStageJob[*corev1.Pod]{
		Resource: pod,
		Stage:    stage,
		Key:      key,
	}
	ok = c.delayQueue.AddAfter(item, delay)
	if !ok {
		logger.Debug("Skip pod",
			"reason", "delayed",
		)
	} else {
		c.delayQueueMapping.Store(key, item)
	}

	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *PodController) playStageWorker(ctx context.Context) {
	for ctx.Err() == nil {
		pod := c.delayQueue.GetOrWait()
		c.delayQueueMapping.Delete(pod.Key)
		c.playStage(ctx, pod.Resource, pod.Stage)
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
func (c *PodController) watchResources(ctx context.Context, events <-chan informer.Event[*corev1.Pod]) {
	logger := log.FromContext(ctx)
loop:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				break loop
			}

			switch event.Type {
			case informer.Added, informer.Modified, informer.Sync:
				pod := event.Object
				if c.enableMetrics {
					c.putPodInfo(pod)
				}
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

				if c.enableMetrics &&
					event.Type == informer.Added &&
					c.nodeGetFunc != nil {
					nodeInfo, ok := c.nodeGetFunc(pod.Spec.NodeName)
					if ok {
						nodeInfo.StartedContainer.Add(int64(len(pod.Spec.Containers)))
					}
				}
			case informer.Deleted:
				pod := event.Object
				if c.enableMetrics {
					c.deletePodInfo(pod)
				}
				if c.need(pod) {
					// Recycling PodIP
					c.recyclingPodIP(ctx, pod)

					// Cancel delay job
					key := log.KObj(pod).String()
					resourceJob, ok := c.delayQueueMapping.LoadAndDelete(key)
					if ok {
						c.delayQueue.Cancel(resourceJob)
					}
				}
			}
		case <-ctx.Done():
			break loop
		}
	}
	logger.Info("Stop watch pods")
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
	_, has := c.nodeGetFunc(pod.Spec.NodeName)
	if !has {
		return
	}

	logger := log.FromContext(ctx)
	if !c.enableCNI {
		if pod.Status.PodIP != "" && c.nodeCacheGetter != nil {
			cidr := c.defaultCIDR
			node, ok := c.nodeCacheGetter.Get(pod.Spec.NodeName)
			if ok {
				if node.Spec.PodCIDR != "" {
					cidr = node.Spec.PodCIDR
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
		if _, has := c.nodeGetFunc(pod.Spec.NodeName); has {
			if pod.Status.PodIP != "" && c.nodeCacheGetter != nil {
				cidr := c.defaultCIDR
				node, ok := c.nodeCacheGetter.Get(pod.Spec.NodeName)
				if ok {
					if node.Spec.PodCIDR != "" {
						cidr = node.Spec.PodCIDR
					}
					pool, err := c.ipPool(cidr)
					if err == nil {
						pool.Use(pod.Status.PodIP)
					}
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
	_, has := c.nodeGetFunc(nodeName)
	if has && c.nodeCacheGetter != nil {
		node, ok := c.nodeCacheGetter.Get(nodeName)
		if ok {
			hostIPs := getNodeHostIPs(node)
			if len(hostIPs) != 0 {
				return hostIPs[0].String()
			}
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
	_, has := c.nodeGetFunc(nodeName)
	if has && c.nodeCacheGetter != nil {
		node, ok := c.nodeCacheGetter.Get(nodeName)
		if ok {
			if node.Spec.PodCIDR != "" {
				podCIDR = node.Spec.PodCIDR
			}
		}
	}

	pool, err := c.ipPool(podCIDR)
	if err == nil {
		return pool.Get(), nil
	}
	return c.nodeIP, nil
}

// putPodInfo puts pod info
func (c *PodController) putPodInfo(pod *corev1.Pod) {
	podInfo := &PodInfo{}
	key := log.KObj(pod)
	c.podsSets.Store(key, podInfo)
	m, ok := c.podsOnNode.Load(pod.Spec.NodeName)
	if !ok {
		m = &maps.SyncMap[log.ObjectRef, *PodInfo]{}
		c.podsOnNode.Store(pod.Spec.NodeName, m)
	}
	m.Store(key, podInfo)
}

// deletePodInfo deletes pod info
func (c *PodController) deletePodInfo(pod *corev1.Pod) {
	key := log.KObj(pod)
	c.podsSets.Delete(key)
	m, ok := c.podsOnNode.Load(pod.Spec.NodeName)
	if !ok {
		return
	}
	m.Delete(key)
}

// Get gets pod info
func (c *PodController) Get(namespace, name string) (*PodInfo, bool) {
	podInfo, ok := c.podsSets.Load(log.KRef(namespace, name))
	if !ok {
		return nil, false
	}
	return podInfo, true
}

// List lists pod info
func (c *PodController) List(nodeName string) ([]log.ObjectRef, bool) {
	m, ok := c.podsOnNode.Load(nodeName)
	if !ok {
		return nil, false
	}
	return m.Keys(), true
}
