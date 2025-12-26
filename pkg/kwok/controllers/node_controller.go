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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	netutils "k8s.io/utils/net"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// NodeController is a fake nodes implementation that can be used to test
type NodeController struct {
	clock                                 clock.Clock
	typedClient                           kubernetes.Interface
	nodeIP                                string
	nodeName                              string
	nodePort                              int
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	onNodeManagedFunc                     func(nodeName string)
	onNodeUnmanagedFunc                   func(nodeName string)
	nodesSets                             maps.SyncMap[string, *NodeInfo]
	renderer                              gotpl.Renderer
	preprocessChan                        chan *corev1.Node
	playStageParallelism                  uint
	lifecycle                             resources.Getter[lifecycle.Lifecycle]
	delayQueue                            queue.WeightDelayingQueue[resourceStageJob[*corev1.Node]]
	delayQueueMapping                     maps.SyncMap[string, resourceStageJob[*corev1.Node]]
	backoff                               wait.Backoff
	recorder                              record.EventRecorder
	readOnlyFunc                          func(nodeName string) bool
	enableMetrics                         bool
}

// NodeControllerConfig is the configuration for the NodeController
type NodeControllerConfig struct {
	Clock                                 clock.Clock
	TypedClient                           kubernetes.Interface
	OnNodeManagedFunc                     func(nodeName string)
	OnNodeUnmanagedFunc                   func(nodeName string)
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	NodeIP                                string
	NodeName                              string
	NodePort                              int
	Lifecycle                             resources.Getter[lifecycle.Lifecycle]
	PlayStageParallelism                  uint
	FuncMap                               gotpl.FuncMap
	Recorder                              record.EventRecorder
	ReadOnlyFunc                          func(nodeName string) bool
	EnableMetrics                         bool
}

// NodeInfo is the collection of necessary node information
type NodeInfo struct {
	StartedContainer atomic.Int64
}

// NewNodeController creates a new fake nodes controller
func NewNodeController(conf NodeControllerConfig) (*NodeController, error) {
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

	c := &NodeController{
		clock:                                 conf.Clock,
		typedClient:                           conf.TypedClient,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		onNodeManagedFunc:                     conf.OnNodeManagedFunc,
		onNodeUnmanagedFunc:                   conf.OnNodeUnmanagedFunc,
		nodeIP:                                conf.NodeIP,
		nodeName:                              conf.NodeName,
		nodePort:                              conf.NodePort,
		delayQueue:                            queue.NewWeightDelayingQueue[resourceStageJob[*corev1.Node]](conf.Clock),
		backoff:                               defaultBackoff(),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *corev1.Node),
		recorder:                              conf.Recorder,
		readOnlyFunc:                          conf.ReadOnlyFunc,
		enableMetrics:                         conf.EnableMetrics,
	}

	funcMap := maps.Merge(gotpl.FuncMap{
		"NodeIP":   c.funcNodeIP,
		"NodeName": c.funcNodeName,
		"NodePort": c.funcNodePort,
	}, conf.FuncMap)
	c.renderer = gotpl.NewRenderer(funcMap)
	return c, nil
}

// Start starts the fake nodes controller
// if nodeSelectorFunc is not nil, it will use it to determine if the node should be managed
func (c *NodeController) Start(ctx context.Context, events <-chan informer.Event[*corev1.Node]) error {
	go c.preprocessWorker(ctx)
	for i := uint(0); i < c.playStageParallelism; i++ {
		go c.playStageWorker(ctx)
	}
	go c.watchResources(ctx, events)
	return nil
}

func (c *NodeController) need(node *corev1.Node) bool {
	if c.disregardStatusWithAnnotationSelector != nil &&
		len(node.Annotations) != 0 &&
		c.disregardStatusWithAnnotationSelector.Matches(labels.Set(node.Annotations)) {
		return false
	}

	if c.disregardStatusWithLabelSelector != nil &&
		len(node.Labels) != 0 &&
		c.disregardStatusWithLabelSelector.Matches(labels.Set(node.Labels)) {
		return false
	}
	return true
}

// ManageNode manages a node
func (c *NodeController) ManageNode(node *corev1.Node) {
	c.preprocessChan <- node
}

// watchResources watch resources and send to preprocessChan
func (c *NodeController) watchResources(ctx context.Context, events <-chan informer.Event[*corev1.Node]) {
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
				node := event.Object
				if c.need(node) {
					c.putNodeInfo(node)
					if c.readOnly(node.Name) {
						logger.Debug("Skip node",
							"reason", "read only",
							"event", event.Type,
							"node", node.Name,
						)
					} else {
						c.preprocessChan <- node
					}

					if c.onNodeManagedFunc != nil && event.Type != informer.Modified {
						c.onNodeManagedFunc(node.Name)
					}
				}
			case informer.Deleted:
				node := event.Object
				if _, has := c.nodesSets.Load(node.Name); has {
					c.deleteNodeInfo(node)

					// Cancel delay job
					key := node.Name
					resourceJob, ok := c.delayQueueMapping.LoadAndDelete(key)
					if ok {
						c.delayQueue.Cancel(resourceJob)
					}
				}

				if c.onNodeUnmanagedFunc != nil {
					c.onNodeUnmanagedFunc(node.Name)
				}
			}
		case <-ctx.Done():
			break loop
		}
	}
	logger.Info("Stop watch nodes")
}

// deleteResource deletes a node
func (c *NodeController) deleteResource(ctx context.Context, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)

	err := c.typedClient.CoreV1().Nodes().Delete(ctx, node.Name, deleteOpt)
	if err != nil {
		return err
	}

	logger.Info("Delete node")
	return nil
}

// preprocessWorker receives the resource from the preprocessChan and preprocess it
func (c *NodeController) preprocessWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop preprocess worker")
			return
		case node := <-c.preprocessChan:
			err := c.preprocess(ctx, node)
			if err != nil {
				logger.Error("Failed to preprocess node", err,
					"node", node.Name,
				)
			}
		}
	}
}

// preprocess the node and send it to the playStageWorker
func (c *NodeController) preprocess(ctx context.Context, node *corev1.Node) error {
	key := node.Name

	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", key,
	)

	resourceJob, ok := c.delayQueueMapping.Load(key)
	if ok {
		if resourceJob.Resource.ResourceVersion == node.ResourceVersion {
			logger.Debug("Skip node",
				"reason", "resource version not changed",
				"stage", resourceJob.Stage.Name(),
			)
			return nil
		}
	}

	lc := c.lifecycle.Get()

	event := &lifecycle.Event{
		Labels:      node.Labels,
		Annotations: node.Annotations,
		Data:        node,
	}
	stage, err := lc.Match(ctx, event)
	if err != nil {
		return fmt.Errorf("stage match: %w", err)
	}
	if stage == nil {
		logger.Debug("Skip node",
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

	item := resourceStageJob[*corev1.Node]{
		Resource:   node,
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
func (c *NodeController) playStageWorker(ctx context.Context) {
	logger := log.FromContext(ctx)

	for ctx.Err() == nil {
		node, ok := c.delayQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		c.delayQueueMapping.Delete(node.Key)
		remainIndex, err := c.playStage(ctx, node.Resource, node.Stage, int(*node.StepIndex))
		if err != nil {
			logger.Error("failed to apply stage", err,
				"node", node.Key,
				"stage", node.Stage.Name(),
			)
		}
		if remainIndex >= 0 {
			atomic.StoreUint64(node.StepIndex, uint64(remainIndex))

			retryCount := atomic.AddUint64(node.RetryCount, 1) - 1
			logger.Info("retrying for failed job",
				"node", node.Key,
				"stage", node.Stage.Name(),
				"retry", retryCount,
			)
			// for failed jobs, we re-push them into the queue with a lower weight
			// and a backoff period to avoid blocking normal tasks
			retryDelay := backoffDelayByStep(retryCount, c.backoff)
			c.addStageJob(ctx, node, retryDelay, 1)
		}
	}
}

// playStage plays the stage.
// The returned boolean indicates whether the applying action needs to be retried.
func (c *NodeController) playStage(ctx context.Context, node *corev1.Node, stage *lifecycle.Stage, stepIndex int) (int, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
		"stage", stage.Name(),
	)

	var (
		result *corev1.Node
	)

	remainIndex, err := stage.DoSteps(
		stepIndex, node.Finalizers, node, c.renderer,
		func(event *internalversion.StageEvent) error {
			if c.recorder != nil {
				c.recorder.Event(&corev1.ObjectReference{
					Kind:      "Node",
					UID:       node.UID,
					Name:      node.Name,
					Namespace: "",
				}, event.Type, event.Reason, event.Message)
			}
			return nil
		},
		func() error {
			err := c.deleteResource(ctx, node)
			if err != nil {
				return fmt.Errorf("failed to delete node %s: %w", node.Name, err)
			}
			result = nil
			return nil
		},
		func(patch *lifecycle.Patch) error {
			changed, err := checkNeedPatchWithTyped(node, patch.Data, patch.Type)
			if err != nil {
				return fmt.Errorf("failed to check need patch for node %s: %w", node.Name, err)
			}
			if !changed {
				logger.Debug("Skip node",
					"reason", "do not need to modify",
				)
			} else {
				result, err = c.patchResource(ctx, node, patch)
				if err != nil {
					return fmt.Errorf("failed to patch node %s: %w", node.Name, err)
				}
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
			"reason", "immediateNextStage is true")
		c.preprocessChan <- result
	}
	return -1, nil
}

func (c *NodeController) readOnly(nodeName string) bool {
	if c.readOnlyFunc == nil {
		return false
	}
	return c.readOnlyFunc(nodeName)
}

// patchResource patches the resource
func (c *NodeController) patchResource(ctx context.Context, node *corev1.Node, patch *lifecycle.Patch) (*corev1.Node, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)

	subresource := []string{}
	if patch.Subresource != "" {
		logger = logger.With(
			"subresource", patch.Subresource,
		)
		subresource = []string{patch.Subresource}
	}
	result, err := c.typedClient.CoreV1().Nodes().Patch(ctx, node.Name, patch.Type, patch.Data, metav1.PatchOptions{}, subresource...)
	if err != nil {
		return nil, err
	}
	logger.Info("Patch node")
	return result, nil
}

// putNodeInfo puts node info
func (c *NodeController) putNodeInfo(node *corev1.Node) {
	c.nodesSets.Store(node.Name, &NodeInfo{})
}

// deleteNodeInfo deletes node info
func (c *NodeController) deleteNodeInfo(node *corev1.Node) {
	c.nodesSets.Delete(node.Name)
}

// getNodeHostIPs returns the provided node's IP(s); either a single "primary IP" for the
// node in a single-stack cluster, or a dual-stack pair of IPs in a dual-stack cluster
// (for nodes that actually have dual-stack IPs). Among other things, the IPs returned
// from this function are used as the `.status.PodIPs` values for host-network pods on the
// node, and the first IP is used as the `.status.HostIP` for all pods on the node.
// Copy from https://github.com/kubernetes/kubernetes/blob/1d02d014e8c1f0de84b0b58b2165548182815320/pkg/util/node/node.go#L67-L104
func getNodeHostIPs(node *corev1.Node) []net.IP {
	nodeIPs := []net.IP{}
	// Re-sort the addresses with InternalIPs first and then ExternalIPs
	allIPs := make([]net.IP, 0, len(node.Status.Addresses))
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			ip := netutils.ParseIPSloppy(addr.Address)
			if ip != nil {
				allIPs = append(allIPs, ip)
			}
		}
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			ip := netutils.ParseIPSloppy(addr.Address)
			if ip != nil {
				allIPs = append(allIPs, ip)
			}
		}
	}

	if len(allIPs) > 0 {
		nodeIPs = append(nodeIPs, allIPs[0])
		if len(allIPs) > 1 {
			for i := 1; i < len(allIPs); i++ {
				if netutils.IsIPv6(allIPs[i]) != netutils.IsIPv6(allIPs[0]) {
					nodeIPs = append(nodeIPs, allIPs[i])
					break
				}
			}
		}
	}

	return nodeIPs
}

// Get returns Has bool and node info
func (c *NodeController) Get(nodeName string) (*NodeInfo, bool) {
	nodeInfo, has := c.nodesSets.Load(nodeName)
	if has {
		return nodeInfo, has
	}
	return nil, has
}

// List returns all name of nodes
func (c *NodeController) List() []string {
	return c.nodesSets.Keys()
}

func (c *NodeController) funcNodeIP() string {
	return c.nodeIP
}

func (c *NodeController) funcNodeName() string {
	return c.nodeName
}

func (c *NodeController) funcNodePort() int {
	return c.nodePort
}

// addStageJob adds a stage to be applied into the underlying weight delay queue and the associated helper map
func (c *NodeController) addStageJob(ctx context.Context, job resourceStageJob[*corev1.Node], delay time.Duration, weight int) {
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
