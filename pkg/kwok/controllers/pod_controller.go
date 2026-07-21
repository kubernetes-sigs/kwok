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
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	utilscel "sigs.k8s.io/kwok/pkg/utils/cel"
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
	nodeLocks                             utilsmaps.SyncMap[string, *sync.Mutex]
	deferredQueue                         queue.DelayingQueue[deferredRetryJob]
	deferredRetries                       utilsmaps.SyncMap[log.ObjectRef, deferredRetryState]
	deferredRetryToken                    atomic.Uint64
	contextRetryQueue                     queue.DelayingQueue[log.ObjectRef]
	onPodUpdated                          func(*corev1.Pod)
	onPodDeleted                          func(*corev1.Pod)
}

// PodInfo is the collection of necessary pod information
type PodInfo struct {
	NodeName           string
	Requests           corev1.ResourceList
	Deferred           bool
	DeferredGeneration int64
	Infeasible         bool
}

type deferredRetryJob struct {
	Ref        log.ObjectRef
	Generation int64
	Token      uint64
}

type deferredRetryState struct {
	Generation int64
	Token      uint64
}

const (
	deferredInitialRetryDelay  = 30 * time.Second
	deferredPeriodicRetryDelay = 3 * time.Minute
)

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
	OnPodUpdated                          func(*corev1.Pod)
	OnPodDeleted                          func(*corev1.Pod)
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
		deferredQueue:                         queue.NewDelayingQueue[deferredRetryJob](conf.Clock),
		contextRetryQueue:                     queue.NewDelayingQueue[log.ObjectRef](conf.Clock),
		onPodUpdated:                          conf.OnPodUpdated,
		onPodDeleted:                          conf.OnPodDeleted,
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
	go c.deferredRetryWorker(ctx)
	go c.contextRetryWorker(ctx)
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

func (c *PodController) deferredRetryWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		job, ok := c.deferredQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		state, ok := c.deferredRetries.Load(job.Ref)
		if !ok || state.Generation != job.Generation || state.Token != job.Token {
			continue
		}

		pod, err := c.typedClient.CoreV1().Pods(job.Ref.Namespace).Get(ctx, job.Ref.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				c.deferredRetries.Delete(job.Ref)
				continue
			}
			logger.Warn("Failed to get deferred pod for retry",
				"err", err,
				"pod", job.Ref,
			)
			c.scheduleDeferredRetry(job.Ref, job.Generation, deferredPeriodicRetryDelay)
			continue
		}
		generation, deferred := podDeferredGeneration(pod)
		if !deferred || generation != job.Generation || !c.need(pod) || c.readOnly(pod.Spec.NodeName) {
			c.deferredRetries.Delete(job.Ref)
			continue
		}

		c.scheduleDeferredRetry(job.Ref, job.Generation, deferredPeriodicRetryDelay)
		select {
		case c.preprocessChan <- pod.DeepCopy():
		case <-ctx.Done():
			return
		}
	}
}

func (c *PodController) contextRetryWorker(ctx context.Context) {
	for ctx.Err() == nil {
		ref, ok := c.contextRetryQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		pod, err := c.typedClient.CoreV1().Pods(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		select {
		case c.preprocessChan <- pod.DeepCopy():
		case <-ctx.Done():
			return
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
	if !c.nodeContextAvailable(pod) {
		c.scheduleContextRetry(pod)
		return fmt.Errorf("node %q is not available for pod lifecycle context", pod.Spec.NodeName)
	}

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

	event := c.lifecycleEvent(pod)

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
	if !stage.UsesPodResourceContext() {
		return c.doPlayStage(ctx, pod, stage, stepIndex)
	}

	nodeLock := c.nodeLock(pod.Spec.NodeName)
	nodeLock.Lock()
	defer nodeLock.Unlock()

	latest, err := c.typedClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return -1, nil
		}
		return stepIndex, fmt.Errorf("failed to revalidate pod %s: %w", pod.Name, err)
	}
	if !c.nodeContextAvailable(latest) {
		return stepIndex, fmt.Errorf("node %q is not available while revalidating pod %s", latest.Spec.NodeName, latest.Name)
	}
	latestStage, err := c.lifecycle.Get().Match(ctx, c.lifecycleEvent(latest))
	if err != nil {
		return -1, fmt.Errorf("failed to revalidate stage for pod %s: %w", pod.Name, err)
	}
	if latestStage == nil {
		return -1, nil
	}
	if latestStage.Name() != stage.Name() {
		stepIndex = 0
	}
	return c.doPlayStage(ctx, latest, latestStage, stepIndex)
}

func (c *PodController) doPlayStage(ctx context.Context, pod *corev1.Pod, stage *lifecycle.Stage, stepIndex int) (int, error) {
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
				if c.putPodInfo(result) {
					c.retryDeferredPods(result.Spec.NodeName)
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

func (c *PodController) lifecycleEvent(pod *corev1.Pod) *lifecycle.Event {
	var node *corev1.Node
	if c.nodeCacheGetter != nil {
		node, _ = c.nodeCacheGetter.Get(pod.Spec.NodeName)
	}
	return &lifecycle.Event{
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Data:        pod,
		CELVariables: map[string]any{
			"node":                  node,
			"nodeAllocatedRequests": utilscel.NewResourceList(c.nodeAllocatedRequests(pod.Spec.NodeName)),
		},
	}
}

func (c *PodController) nodeLock(nodeName string) *sync.Mutex {
	lock, _ := c.nodeLocks.LoadOrStore(nodeName, &sync.Mutex{})
	return lock
}

func (c *PodController) nodeContextAvailable(pod *corev1.Pod) bool {
	if c.nodeCacheGetter == nil {
		return true
	}
	_, ok := c.nodeCacheGetter.Get(pod.Spec.NodeName)
	return ok
}

func (c *PodController) scheduleContextRetry(pod *corev1.Pod) {
	c.scheduleContextRetryRef(log.KObj(pod), time.Second)
}

func (c *PodController) scheduleContextRetryRef(ref log.ObjectRef, delay time.Duration) {
	c.contextRetryQueue.Cancel(ref)
	c.contextRetryQueue.AddAfter(ref, delay)
}

func (c *PodController) nodeAllocatedRequests(nodeName string) corev1.ResourceList {
	requests := corev1.ResourceList{}
	pods, ok := c.podsOnNode.Load(nodeName)
	if !ok {
		return requests
	}
	pods.Range(func(_ log.ObjectRef, pod *PodInfo) bool {
		addResourceList(requests, pod.Requests)
		return true
	})
	return requests
}

// NodeAllocatableChanged retries deferred pods after a node capacity change.
func (c *PodController) NodeAllocatableChanged(nodeName string) {
	c.retryDeferredPods(nodeName)
	c.retryInfeasiblePods(nodeName)
}

func (c *PodController) retryDeferredPods(nodeName string) {
	pods, ok := c.podsOnNode.Load(nodeName)
	if !ok {
		return
	}
	pods.Range(func(ref log.ObjectRef, pod *PodInfo) bool {
		if pod.Deferred {
			c.scheduleDeferredRetry(ref, pod.DeferredGeneration, 0)
		}
		return true
	})
}

func (c *PodController) retryInfeasiblePods(nodeName string) {
	pods, ok := c.podsOnNode.Load(nodeName)
	if !ok {
		return
	}
	pods.Range(func(ref log.ObjectRef, pod *PodInfo) bool {
		if pod.Infeasible {
			c.scheduleContextRetryRef(ref, 0)
		}
		return true
	})
}

func (c *PodController) ensureDeferredRetry(pod *corev1.Pod, generation int64) {
	ref := log.KObj(pod)
	state, ok := c.deferredRetries.Load(ref)
	if ok && state.Generation == generation {
		return
	}
	c.scheduleDeferredRetry(ref, generation, deferredInitialRetryDelay)
}

func (c *PodController) scheduleDeferredRetry(ref log.ObjectRef, generation int64, delay time.Duration) {
	if state, ok := c.deferredRetries.Load(ref); ok {
		c.deferredQueue.Cancel(deferredRetryJob{Ref: ref, Generation: state.Generation, Token: state.Token})
	}
	token := c.deferredRetryToken.Add(1)
	c.deferredRetries.Store(ref, deferredRetryState{Generation: generation, Token: token})
	c.deferredQueue.AddAfter(deferredRetryJob{Ref: ref, Generation: generation, Token: token}, delay)
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
				allocationDecreased := c.putPodInfo(pod)
				if allocationDecreased {
					c.retryDeferredPods(pod.Spec.NodeName)
				}
				if c.need(pod) {
					if c.readOnly(pod.Spec.NodeName) {
						logger.Debug("Skip pod",
							"reason", "read only",
							"event", event.Type,
							"pod", log.KObj(pod),
							"node", pod.Spec.NodeName,
						)
					} else if generation, deferred := podDeferredGeneration(pod); deferred {
						c.ensureDeferredRetry(pod, generation)
					} else {
						c.deferredRetries.Delete(log.KObj(pod))
						c.preprocessChan <- pod.DeepCopy()
					}
				} else {
					c.deferredRetries.Delete(log.KObj(pod))
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
				allocationReleased := c.deletePodInfo(pod)
				c.deferredRetries.Delete(log.KObj(pod))
				c.contextRetryQueue.Cancel(log.KObj(pod))
				if allocationReleased {
					c.retryDeferredPods(pod.Spec.NodeName)
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
	return c.nodeIP
}

func (c *PodController) nodeIPs() []string {
	if c.nodeIP == "" {
		return []string{}
	}
	return []string{c.nodeIP}
}

func (c *PodController) funcNodeIPWith(nodeName string) string {
	if c.nodeCacheGetter == nil {
		return c.nodeIP
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.nodeIP
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.nodeIP
	}

	hostIPs := getNodeHostIPs(node)
	if len(hostIPs) != 0 {
		return hostIPs[0].String()
	}

	return c.nodeIP
}

func (c *PodController) funcNodeIPsWith(nodeName string) []string {
	if c.nodeCacheGetter == nil {
		return c.nodeIPs()
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.nodeIPs()
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.nodeIPs()
	}

	hostIPs := getNodeHostIPs(node)
	if len(hostIPs) != 0 {
		return utilsslices.Map(hostIPs, func(ip net.IP) string {
			return ip.String()
		})
	}

	return c.nodeIPs()
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

	if c.nodeCacheGetter == nil {
		return c.nodeIP, nil
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.nodeIP, nil
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.nodeIP, nil
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
	return c.nodeIP, nil
}

func (c *PodController) funcPodIPsWith(nodeName string, hostNetwork bool, uid, name, namespace string) ([]string, error) {
	if hostNetwork {
		return c.funcNodeIPsWith(nodeName), nil
	}

	if c.nodeCacheGetter == nil {
		return c.nodeIPs(), nil
	}

	_, has := c.nodeGetFunc(nodeName)
	if !has {
		return c.nodeIPs(), nil
	}

	node, ok := c.nodeCacheGetter.Get(nodeName)
	if !ok {
		return c.nodeIPs(), nil
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
			continue
		}
		ips = append(ips, pool.Get())
	}
	if len(ips) == 0 {
		return c.nodeIPs(), nil
	}
	return ips, nil
}

// putPodInfo puts pod info
func (c *PodController) putPodInfo(pod *corev1.Pod) bool {
	if c.onPodUpdated != nil {
		c.onPodUpdated(pod)
	}
	deferredGeneration, deferred := podDeferredGeneration(pod)
	podInfo := &PodInfo{
		NodeName:           pod.Spec.NodeName,
		Requests:           podAllocatedRequests(pod),
		Deferred:           deferred,
		DeferredGeneration: deferredGeneration,
		Infeasible:         podInfeasible(pod),
	}
	key := log.KObj(pod)
	old, loaded := c.podsSets.Swap(key, podInfo)
	if loaded && old.NodeName != podInfo.NodeName {
		if oldNodePods, ok := c.podsOnNode.Load(old.NodeName); ok {
			oldNodePods.Delete(key)
		}
		c.retryDeferredPods(old.NodeName)
	}
	m, ok := c.podsOnNode.Load(pod.Spec.NodeName)
	if !ok {
		m = &utilsmaps.SyncMap[log.ObjectRef, *PodInfo]{}
		c.podsOnNode.Store(pod.Spec.NodeName, m)
	}
	m.Store(key, podInfo)
	return loaded && (old.NodeName != podInfo.NodeName || resourceListDecreased(old.Requests, podInfo.Requests))
}

func podAllocatedRequests(pod *corev1.Pod) corev1.ResourceList {
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return corev1.ResourceList{}
	}
	return utilscel.PodRequests(pod, true)
}

func podDeferredGeneration(pod *corev1.Pod) (int64, bool) {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodResizePending &&
			condition.Status == corev1.ConditionTrue &&
			condition.Reason == corev1.PodReasonDeferred &&
			condition.ObservedGeneration == pod.Generation {
			return condition.ObservedGeneration, true
		}
	}
	return 0, false
}

func podInfeasible(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodResizePending &&
			condition.Status == corev1.ConditionTrue &&
			condition.Reason == corev1.PodReasonInfeasible &&
			condition.ObservedGeneration == pod.Generation {
			return true
		}
	}
	return false
}

func addResourceList(target, resources corev1.ResourceList) {
	for name, quantity := range resources {
		current := target[name]
		current.Add(quantity)
		target[name] = current
	}
}

func resourceListDecreased(old, current corev1.ResourceList) bool {
	for name, oldQuantity := range old {
		if oldQuantity.Cmp(current[name]) > 0 {
			return true
		}
	}
	return false
}

func resourceListsEqual(a, b corev1.ResourceList) bool {
	if len(a) != len(b) {
		return false
	}
	for name, quantity := range a {
		other, ok := b[name]
		if !ok || !quantity.Equal(other) {
			return false
		}
	}
	return true
}

// deletePodInfo deletes pod info
func (c *PodController) deletePodInfo(pod *corev1.Pod) bool {
	if c.onPodDeleted != nil {
		c.onPodDeleted(pod)
	}
	key := log.KObj(pod)
	podInfo, loaded := c.podsSets.LoadAndDelete(key)
	if !loaded {
		return false
	}
	m, ok := c.podsOnNode.Load(podInfo.NodeName)
	if !ok {
		return len(podInfo.Requests) != 0
	}
	m.Delete(key)
	return len(podInfo.Requests) != 0
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
