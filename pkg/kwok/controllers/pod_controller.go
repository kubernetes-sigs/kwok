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
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	utilsmaps "sigs.k8s.io/kwok/pkg/utils/maps"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

var (
	deleteOpt = *metav1.NewDeleteOptions(0)
)

// PodController is a fake pods implementation that can be used to test
type PodController struct {
	clock                                 clock.Clock
	dynamicClient                         dynamic.Interface
	restMapper                            meta.RESTMapper
	typedClient                           kubernetes.Interface
	impersonatingTypedClient              client.TypedClientImpersonator
	nodeCacheGetter                       informer.Getter[*corev1.Node]
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	nodeIP                                string
	defaultCIDR                           string
	nodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	ipPools                               utilsmaps.SyncMap[string, *ipPool]
	renderer                              gotpl.Renderer
	podsSets                              utilsmaps.SyncMap[log.ObjectRef, *PodInfo]
	podsOnNode                            utilsmaps.SyncMap[string, *utilsmaps.SyncMap[log.ObjectRef, *PodInfo]]
	preprocessChan                        chan *corev1.Pod
	playStageParallelism                  uint
	lifecycle                             resources.Getter[lifecycle.Lifecycle]
	delayQueue                            queue.WeightDelayingQueue[resourceStageJob[*corev1.Pod]]
	backoff                               wait.Backoff
	delayQueueMapping                     utilsmaps.SyncMap[string, resourceStageJob[*corev1.Pod]]
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
	DynamicClient                         dynamic.Interface
	RESTMapper                            meta.RESTMapper
	TypedClient                           kubernetes.Interface
	ImpersonatingTypedClient              client.TypedClientImpersonator
	NodeCacheGetter                       informer.Getter[*corev1.Node]
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	NodeIP                                string
	CIDR                                  string
	NodeGetFunc                           func(nodeName string) (*NodeInfo, bool)
	NodeHasMetric                         func(nodeName string) bool
	Lifecycle                             resources.Getter[lifecycle.Lifecycle]
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
		dynamicClient:                         conf.DynamicClient,
		restMapper:                            conf.RESTMapper,
		typedClient:                           conf.TypedClient,
		impersonatingTypedClient:              conf.ImpersonatingTypedClient,
		nodeCacheGetter:                       conf.NodeCacheGetter,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		nodeIP:                                conf.NodeIP,
		defaultCIDR:                           conf.CIDR,
		nodeGetFunc:                           conf.NodeGetFunc,
		delayQueue:                            queue.NewWeightDelayingQueue[resourceStageJob[*corev1.Pod]](conf.Clock),
		backoff:                               defaultBackoff(),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *corev1.Pod),
		recorder:                              conf.Recorder,
		readOnlyFunc:                          conf.ReadOnlyFunc,
		enableMetrics:                         conf.EnableMetrics,
	}
	funcMap := utilsmaps.Merge(gotpl.FuncMap{
		"NodeIP":      c.funcNodeIP,
		"PodIP":       c.funcPodIP,
		"NodeIPWith":  c.funcNodeIPWith,
		"NodeIPsWith": c.funcNodeIPsWith,
		"PodIPWith":   c.funcPodIPWith,
		"PodIPsWith":  c.funcPodIPsWith,
	}, conf.FuncMap)
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

// deleteResource deletes a pod
func (c *PodController) deleteResource(ctx context.Context, pod *corev1.Pod) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	err := c.typedClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOpt)
	if err != nil {
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
				logger.Error("Failed to preprocess node",
					"err", err,
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

	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", key,
		"node", pod.Spec.NodeName,
	)

	resourceJob, ok := c.delayQueueMapping.Load(key)
	if ok {
		if resourceJob.Resource.ResourceVersion == pod.ResourceVersion {
			logger.Debug("Skip pod",
				"reason", "resource version not changed",
				"stage", resourceJob.Stage.Name(),
			)
			return nil
		}
	}

	event := &lifecycle.Event{
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Data:        pod,
	}

	lc := c.lifecycle.Get()
	stage, err := lc.Match(ctx, event)
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
	delay, _, err := stage.Delay(ctx, event, now)
	if err != nil {
		logger.Warn("Failed to get delay",
			"stage", stage.Name(),
			"err", err,
		)
	}
	if delay != 0 {
		stageName := stage.Name()
		logger.Debug("Delayed play stage",
			"delay", delay,
			"stage", stageName,
		)
	}

	item := resourceStageJob[*corev1.Pod]{
		Resource:   pod,
		Stage:      stage,
		Key:        key,
		RetryCount: new(uint64),
		StepIndex:  new(uint64),
	}
	// we add a normal(fresh) stage job with weight 0,
	// resulting in that it will always be processed with high priority compared to those retry ones
	c.addStageJob(ctx, item, delay, 0)
	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *PodController) playStageWorker(ctx context.Context) {
	logger := log.FromContext(ctx)

	for ctx.Err() == nil {
		pod, ok := c.delayQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		c.delayQueueMapping.Delete(pod.Key)
		remainIndex, err := c.playStage(ctx, pod.Resource, pod.Stage, int(*pod.StepIndex))
		if err != nil {
			logger.Error("failed to apply stage",
				"err", err,
				"pod", pod.Key,
				"stage", pod.Stage.Name(),
			)
		}
		if remainIndex >= 0 {
			atomic.StoreUint64(pod.StepIndex, uint64(remainIndex))

			retryCount := atomic.AddUint64(pod.RetryCount, 1) - 1
			logger.Info("retrying for failed job",
				"pod", pod.Key,
				"stage", pod.Stage.Name(),
				"retry", retryCount,
			)
			// for failed jobs, we re-push them into the queue with a lower weight
			// and a backoff period to avoid blocking normal tasks
			retryDelay := backoffDelayByStep(retryCount, c.backoff)
			c.addStageJob(ctx, pod, retryDelay, 1)
		}
	}
}

// playStage plays the stage.
// The returned boolean indicates whether the applying action needs to be retried.
func (c *PodController) playStage(ctx context.Context, pod *corev1.Pod, stage *lifecycle.Stage, stepIndex int) (int, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
		"stage", stage.Name(),
	)

	var (
		result *corev1.Pod
	)

	remainIndex, err := stage.DoSteps(
		stepIndex, pod.Finalizers, pod, c.renderer,
		func(event *internalversion.StageEvent) error {
			if c.recorder != nil {
				c.recorder.Event(&corev1.ObjectReference{
					Kind:      "Pod",
					UID:       pod.UID,
					Name:      pod.Name,
					Namespace: pod.Namespace,
				}, event.Type, event.Reason, event.Message)
			}
			return nil
		},
		func() error {
			err := c.deleteResource(ctx, pod)
			if err != nil {
				return fmt.Errorf("failed to delete pod %s: %w", pod.Name, err)
			}
			result = nil
			return nil
		},
		func(patch *lifecycle.Patch) error {
			c.markPodIP(ctx, pod)
			changed, err := checkNeedPatchWithTyped(pod, patch.Data, patch.Type)
			if err != nil {
				return fmt.Errorf("failed to check need patch for pod %s: %w", pod.Name, err)
			}
			if !changed {
				logger.Debug("Skip pod",
					"reason", "do not need to modify",
				)
			} else {
				result, err = c.patchResource(ctx, pod, patch)
				if err != nil {
					return fmt.Errorf("failed to patch pod %s: %w", pod.Name, err)
				}
			}
			return nil
		},
		func(apply *lifecycle.Apply) error {
			err := c.applyResource(ctx, pod, apply)
			if err != nil {
				return fmt.Errorf("failed to apply resource: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		if shouldRetry(err) {
			return remainIndex, err
		}
		return -1, err
	}

	if result != nil && stage.ImmediateNextStage() {
		logger.Debug("Re-push to preprocessChan",
			"reason", "immediateNextStage is true",
		)
		c.preprocessChan <- result
	}
	return -1, nil
}

func (c *PodController) readOnly(nodeName string) bool {
	if c.readOnlyFunc == nil {
		return false
	}
	return c.readOnlyFunc(nodeName)
}

// applyResource applies the resource
func (c *PodController) applyResource(ctx context.Context, pod *corev1.Pod, apply *lifecycle.Apply) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	_, err := lifecycle.ApplyResource(
		ctx,
		c.dynamicClient,
		c.restMapper,
		pod.Namespace,
		apply,
	)
	if err != nil {
		return err
	}
	logger.Info("Apply resource")
	return nil
}

// patchResource patches the resource
func (c *PodController) patchResource(ctx context.Context, pod *corev1.Pod, patch *lifecycle.Patch) (*corev1.Pod, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"pod", log.KObj(pod),
		"node", pod.Spec.NodeName,
	)

	subresource := []string{}
	if patch.Subresource != "" {
		logger = logger.With(
			"subresource", patch.Subresource,
		)
		subresource = []string{patch.Subresource}
	}

	client := c.typedClient

	if patch.Impersonation != nil {
		if c.impersonatingTypedClient == nil {
			return nil, fmt.Errorf("impersonating typed client is not configured")
		}

		logger = logger.With(
			"impersonate", patch.Impersonation.Username,
		)

		tc, err := c.impersonatingTypedClient.ImpersonateTyped(rest.ImpersonationConfig{UserName: patch.Impersonation.Username})
		if err != nil {
			return nil, err
		}

		client = tc
	}

	result, err := client.CoreV1().Pods(pod.Namespace).Patch(ctx, pod.Name, patch.Type, patch.Data, metav1.PatchOptions{}, subresource...)
	if err != nil {
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
	ipnet, err := utilsnet.ParseCIDR(cidr)
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

	if c.nodeCacheGetter == nil {
		return
	}

	_, has := c.nodeGetFunc(pod.Spec.NodeName)
	if !has {
		return
	}

	node, ok := c.nodeCacheGetter.Get(pod.Spec.NodeName)
	if !ok {
		return
	}

	cidrs := []string{c.defaultCIDR}
	if len(node.Spec.PodCIDRs) != 0 {
		cidrs = node.Spec.PodCIDRs
	} else if node.Spec.PodCIDR != "" {
		cidrs = []string{node.Spec.PodCIDR}
	}

	ips := make([]string, 0, min(len(pod.Status.PodIPs), 1))
	if len(pod.Status.PodIPs) != 0 {
		for _, podIP := range pod.Status.PodIPs {
			ips = append(ips, podIP.IP)
		}
	} else if pod.Status.PodIP != "" {
		ips = []string{pod.Status.PodIP}
	}

	for i, ip := range ips {
		if i >= len(cidrs) {
			break
		}
		pool, err := c.ipPool(cidrs[i])
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Error("Failed to get ip pool",
				"err", err,
				"pod", log.KObj(pod),
				"node", pod.Spec.NodeName,
			)
		} else {
			pool.Put(ip)
		}
	}
}

func (c *PodController) markPodIP(ctx context.Context, pod *corev1.Pod) {
	// Skip host network
	if pod.Spec.HostNetwork {
		return
	}

	if c.nodeCacheGetter == nil {
		return
	}

	_, has := c.nodeGetFunc(pod.Spec.NodeName)
	if !has {
		return
	}

	node, ok := c.nodeCacheGetter.Get(pod.Spec.NodeName)
	if !ok {
		return
	}

	cidrs := []string{c.defaultCIDR}
	if len(node.Spec.PodCIDRs) != 0 {
		cidrs = node.Spec.PodCIDRs
	} else if node.Spec.PodCIDR != "" {
		cidrs = []string{node.Spec.PodCIDR}
	}

	ips := make([]string, 0, min(len(pod.Status.PodIPs), 1))
	if len(pod.Status.PodIPs) != 0 {
		for _, podIP := range pod.Status.PodIPs {
			ips = append(ips, podIP.IP)
		}
	} else if pod.Status.PodIP != "" {
		ips = []string{pod.Status.PodIP}
	}

	// Mark the pod IP that existed before the kubelet was started

	for i, ip := range ips {
		if i >= len(cidrs) {
			break
		}
		pool, err := c.ipPool(cidrs[i])
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Error("Failed to get ip pool",
				"err", err,
				"pod", log.KObj(pod),
				"node", pod.Spec.NodeName,
			)
		} else {
			pool.Use(ip)
		}
	}
}

func (c *PodController) funcNodeIP() string {
	if c.nodeIP == "" {
		return ""
	}
	return c.nodeIP
}

func (c *PodController) funcNodeIPWith(nodeName string) string {
	if c.nodeCacheGetter == nil {
		return c.funcNodeIP()
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.funcNodeIP()
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.funcNodeIP()
	}

	hostIPs := getNodeHostIPs(node)
	if len(hostIPs) != 0 {
		return hostIPs[0].String()
	}

	return c.funcNodeIP()
}

func (c *PodController) funcNodeIPsWith(nodeName string) []string {
	if c.nodeCacheGetter == nil {
		return []string{c.nodeIP}
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return []string{c.nodeIP}
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return []string{c.nodeIP}
	}

	hostIPs := getNodeHostIPs(node)
	if len(hostIPs) != 0 {
		return utilsslices.Map(hostIPs, func(ip net.IP) string {
			return ip.String()
		})
	}

	return []string{c.nodeIP}
}

func (c *PodController) funcPodIP() string {
	podCIDR := c.defaultCIDR
	pool, err := c.ipPool(podCIDR)
	if err == nil {
		return pool.Get()
	}
	return c.funcNodeIP()
}

func (c *PodController) funcPodIPWith(nodeName string, hostNetwork bool, uid, name, namespace string) (string, error) {
	if hostNetwork {
		return c.funcNodeIPWith(nodeName), nil
	}

	if c.nodeCacheGetter == nil {
		return c.funcNodeIP(), nil
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.funcNodeIP(), nil
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.funcNodeIP(), nil
	}

	podCIDRs := []string{c.defaultCIDR}
	if len(node.Spec.PodCIDRs) != 0 {
		podCIDRs = node.Spec.PodCIDRs
	} else if node.Spec.PodCIDR != "" {
		podCIDRs = []string{node.Spec.PodCIDR}
	}

	pool, err := c.ipPool(podCIDRs[0])
	if err == nil {
		return pool.Get(), nil
	}
	return c.funcNodeIP(), nil
}

func (c *PodController) funcPodIPsWith(nodeName string, hostNetwork bool, uid, name, namespace string) ([]string, error) {
	if hostNetwork {
		return c.funcNodeIPsWith(nodeName), nil
	}

	if c.nodeCacheGetter == nil {
		return []string{c.nodeIP}, nil
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return []string{c.nodeIP}, nil
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return []string{c.nodeIP}, nil
	}

	podCIDRs := []string{c.defaultCIDR}
	if len(node.Spec.PodCIDRs) != 0 {
		podCIDRs = node.Spec.PodCIDRs
	} else if node.Spec.PodCIDR != "" {
		podCIDRs = []string{node.Spec.PodCIDR}
	}

	ips := make([]string, 0, len(podCIDRs))
	for _, podCIDR := range podCIDRs {
		pool, err := c.ipPool(podCIDR)
		if err != nil {
			logger := log.FromContext(context.Background())
			logger.Error("Failed to get ip pool",
				"err", err,
				"node", nodeName,
				"podCIDR", podCIDR,
			)
			continue
		}
		ips = append(ips, pool.Get())
	}
	if len(ips) == 0 {
		return []string{c.nodeIP}, nil
	}
	return ips, nil
}

// putPodInfo puts pod info
func (c *PodController) putPodInfo(pod *corev1.Pod) {
	podInfo := &PodInfo{}
	key := log.KObj(pod)
	c.podsSets.Store(key, podInfo)
	m, ok := c.podsOnNode.Load(pod.Spec.NodeName)
	if !ok {
		m = &utilsmaps.SyncMap[log.ObjectRef, *PodInfo]{}
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

// addStageJob adds a stage to be applied into the underlying weight delay queue and the associated helper map
func (c *PodController) addStageJob(ctx context.Context, job resourceStageJob[*corev1.Pod], delay time.Duration, weight int) {
	old, loaded := c.delayQueueMapping.Swap(job.Key, job)
	if loaded {
		if !c.delayQueue.Cancel(old) {
			logger := log.FromContext(ctx)
			logger.Debug("Failed to cancel stage",
				"stage", job.Stage.Name(),
			)
		}
	}
	c.delayQueue.AddWeightAfter(job, weight, delay)
}
