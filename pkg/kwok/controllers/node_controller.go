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
	"sync/atomic"
	"time"

	"github.com/wzshiming/cron"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/pager"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	netutils "k8s.io/utils/net"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

var (

	// https://kubernetes.io/docs/concepts/architecture/nodes/#condition
	nodeConditions = []corev1.NodeCondition{
		{
			Type:    corev1.NodeReady,
			Status:  corev1.ConditionTrue,
			Reason:  "KubeletReady",
			Message: "kubelet is posting ready status",
		},
		{
			Type:    corev1.NodeMemoryPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasSufficientMemory",
			Message: "kubelet has sufficient memory available",
		},
		{
			Type:    corev1.NodeDiskPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasNoDiskPressure",
			Message: "kubelet has no disk pressure",
		},
		{
			Type:    corev1.NodePIDPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasSufficientPID",
			Message: "kubelet has sufficient PID available",
		},
		{
			Type:    corev1.NodeNetworkUnavailable,
			Status:  corev1.ConditionFalse,
			Reason:  "RouteCreated",
			Message: "RouteController created a route",
		},
	}
	nodeConditionsData, _ = expression.ToJSONStandard(nodeConditions)
	nodeKind              = corev1.SchemeGroupVersion.WithKind("Node")
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
	manageNodesWithLabelSelector          string
	nodeSelectorFunc                      func(node *corev1.Node) bool
	onNodeManagedFunc                     func(nodeName string)
	nodesSets                             maps.SyncMap[string, *NodeInfo]
	renderer                              gotpl.Renderer
	preprocessChan                        chan *corev1.Node
	playStageChan                         chan resourceStageJob[*corev1.Node]
	playStageParallelism                  uint
	lifecycle                             resources.Getter[Lifecycle]
	cronjob                               *cron.Cron
	delayJobs                             jobInfoMap
	recorder                              record.EventRecorder
	readOnlyFunc                          func(nodeName string) bool
	triggerPreprocessChan                 chan string
	enableMetrics                         bool
}

// NodeControllerConfig is the configuration for the NodeController
type NodeControllerConfig struct {
	Clock                                 clock.Clock
	TypedClient                           kubernetes.Interface
	NodeSelectorFunc                      func(node *corev1.Node) bool
	OnNodeManagedFunc                     func(nodeName string)
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	ManageNodesWithLabelSelector          string
	NodeIP                                string
	NodeName                              string
	NodePort                              int
	Lifecycle                             resources.Getter[Lifecycle]
	PlayStageParallelism                  uint
	FuncMap                               gotpl.FuncMap
	Recorder                              record.EventRecorder
	ReadOnlyFunc                          func(nodeName string) bool
	EnableMetrics                         bool
}

// NodeInfo is the collection of necessary node information
type NodeInfo struct {
	Node             *corev1.Node
	HostIPs          []string
	PodCIDRs         []string
	OwnerReferences  []metav1.OwnerReference
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
		nodeSelectorFunc:                      conf.NodeSelectorFunc,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		manageNodesWithLabelSelector:          conf.ManageNodesWithLabelSelector,
		onNodeManagedFunc:                     conf.OnNodeManagedFunc,
		nodeIP:                                conf.NodeIP,
		nodeName:                              conf.NodeName,
		nodePort:                              conf.NodePort,
		cronjob:                               cron.NewCron(),
		lifecycle:                             conf.Lifecycle,
		playStageParallelism:                  conf.PlayStageParallelism,
		preprocessChan:                        make(chan *corev1.Node),
		triggerPreprocessChan:                 make(chan string, 16),
		playStageChan:                         make(chan resourceStageJob[*corev1.Node]),
		recorder:                              conf.Recorder,
		readOnlyFunc:                          conf.ReadOnlyFunc,
		enableMetrics:                         conf.EnableMetrics,
	}
	funcMap := gotpl.FuncMap{
		"NodeIP":   c.funcNodeIP,
		"NodeName": c.funcNodeName,
		"NodePort": c.funcNodePort,
		"NodeConditions": func() interface{} {
			return nodeConditionsData
		},
	}
	for k, v := range conf.FuncMap {
		funcMap[k] = v
	}
	c.renderer = gotpl.NewRenderer(funcMap)
	return c, nil
}

// Start starts the fake nodes controller
// if nodeSelectorFunc is not nil, it will use it to determine if the node should be managed
func (c *NodeController) Start(ctx context.Context) error {
	go c.preprocessWorker(ctx)
	go c.triggerPreprocessWorker(ctx)
	for i := uint(0); i < c.playStageParallelism; i++ {
		go c.playStageWorker(ctx)
	}

	opt := metav1.ListOptions{
		LabelSelector: c.manageNodesWithLabelSelector,
	}
	err := c.watchResources(ctx, opt)
	if err != nil {
		return fmt.Errorf("failed watch nodes: %w", err)
	}

	logger := log.FromContext(ctx)
	go func() {
		err = c.listResources(ctx, opt)
		if err != nil {
			logger.Error("Failed list nodes", err)
		}
	}()
	return nil
}

func (c *NodeController) need(node *corev1.Node) bool {
	if !c.nodeSelectorFunc(node) {
		return false
	}
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

// watchResources watch resources and send to preprocessChan
func (c *NodeController) watchResources(ctx context.Context, opt metav1.ListOptions) error {
	// Watch nodes in the cluster
	watcher, err := c.typedClient.CoreV1().Nodes().Watch(ctx, opt)
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
						watcher, err := c.typedClient.CoreV1().Nodes().Watch(ctx, opt)
						if err == nil {
							rc = watcher.ResultChan()
							continue loop
						}

						logger.Error("Failed to watch nodes", err)
						select {
						case <-ctx.Done():
							break loop
						case <-c.clock.After(time.Second * 5):
						}
					}
				}
				switch event.Type {
				case watch.Added:
					node := event.Object.(*corev1.Node)
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
						if c.onNodeManagedFunc != nil {
							c.onNodeManagedFunc(node.Name)
						}
					}
				case watch.Modified:
					node := event.Object.(*corev1.Node)
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
					}
				case watch.Deleted:
					node := event.Object.(*corev1.Node)
					if _, has := c.nodesSets.Load(node.Name); has {
						c.deleteNodeInfo(node)

						// Cancel delay job
						key := node.Name
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
		logger.Info("Stop watch nodes")
	}()
	return nil
}

// listResources lists all resources and sends to preprocessChan
func (c *NodeController) listResources(ctx context.Context, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.typedClient.CoreV1().Nodes().List(ctx, opts)
	})

	logger := log.FromContext(ctx)

	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		node := obj.(*corev1.Node)
		if c.need(node) {
			c.putNodeInfo(node)
			if c.readOnly(node.Name) {
				logger.Debug("Skip node",
					"node", node.Name,
					"reason", "read only",
				)
			} else {
				c.preprocessChan <- node
			}
		}
		return nil
	})
}

// finalizersModify modify finalizers of node
func (c *NodeController) finalizersModify(ctx context.Context, node *corev1.Node, finalizers *internalversion.StageFinalizers) (*corev1.Node, error) {
	ops := finalizersModify(node.Finalizers, finalizers)
	if len(ops) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return nil, err
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)

	result, err := c.typedClient.CoreV1().Nodes().Patch(ctx, node.Name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch node finalizers",
				"err", err,
			)
			return nil, nil
		}
		return nil, err
	}
	logger.Info("Patch node finalizers")
	return result, nil
}

// deleteResource deletes a node
func (c *NodeController) deleteResource(ctx context.Context, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)

	err := c.typedClient.CoreV1().Nodes().Delete(ctx, node.Name, deleteOpt)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Delete node",
				"err", err,
			)
			return nil
		}
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

// triggerPreprocessWorker receives the resource from the triggerPreprocessChan and preprocess it
func (c *NodeController) triggerPreprocessWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop trigger preprocess worker")
			return
		case nodeName := <-c.triggerPreprocessChan:
			nodeInfo, has := c.nodesSets.Load(nodeName)
			if !has || nodeInfo.Node == nil {
				logger.Warn("Node not found",
					"node", nodeName,
				)
				continue
			}
			if c.readOnly(nodeInfo.Node.Name) {
				logger.Debug("Skip node",
					"node", nodeInfo.Node.Name,
					"reason", "read only",
				)
			} else {
				c.preprocessChan <- nodeInfo.Node
			}
		}
	}
}

// preprocess the pod and send it to the playStageWorker
func (c *NodeController) preprocess(ctx context.Context, node *corev1.Node) error {
	key := node.Name

	resourceJob, ok := c.delayJobs.Load(key)
	if ok && resourceJob.ResourceVersion == node.ResourceVersion {
		return nil
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", key,
	)

	data, err := expression.ToJSONStandard(node)
	if err != nil {
		return err
	}

	lifecycle := c.lifecycle.Get()
	stage, err := lifecycle.Match(node.Labels, node.Annotations, data)
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
		c.playStageChan <- resourceStageJob[*corev1.Node]{
			Resource: node,
			Stage:    stage,
		}
	})
	if ok {
		resourceJob, ok := c.delayJobs.LoadOrStore(key, jobInfo{
			ResourceVersion: node.ResourceVersion,
			Cancel:          cancelFunc,
		})
		if ok {
			resourceJob.Cancel()
		}
	}
	return nil
}

// playStageWorker receives the resource from the playStageChan and play the stage
func (c *NodeController) playStageWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop play stage worker")
			return
		case node := <-c.playStageChan:
			c.playStage(ctx, node.Resource, node.Stage)
		}
	}
}

// playStage plays the stage
func (c *NodeController) playStage(ctx context.Context, node *corev1.Node, stage *LifecycleStage) {
	next := stage.Next()
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
		"stage", stage.Name(),
	)

	if next.Event != nil && c.recorder != nil {
		c.recorder.Event(&corev1.ObjectReference{
			Kind:      "Node",
			UID:       node.UID,
			Name:      node.Name,
			Namespace: "",
		}, next.Event.Type, next.Event.Reason, next.Event.Message)
	}
	if next.Finalizers != nil {
		result, err := c.finalizersModify(ctx, node, next.Finalizers)
		if err != nil {
			logger.Error("Failed to finalizers of node", err)
		}
		if result != nil && stage.ImmediateNextStage() {
			c.preprocessChan <- result
		}
	}
	if next.Delete {
		err := c.deleteResource(ctx, node)
		if err != nil {
			logger.Error("Failed to delete node", err)
		}
	} else if next.StatusTemplate != "" {
		patch, err := c.computePatch(node, next.StatusTemplate)
		if err != nil {
			logger.Error("Failed to configure node", err)
			return
		}
		if patch == nil {
			logger.Debug("Skip node",
				"reason", "do not need to modify",
			)
		} else {
			result, err := c.patchResource(ctx, node, patch)
			if err != nil {
				logger.Error("Failed to patch node", err)
			}
			if result != nil && stage.ImmediateNextStage() {
				c.preprocessChan <- result
			}
		}
	}
}

func (c *NodeController) readOnly(nodeName string) bool {
	if c.readOnlyFunc == nil {
		return false
	}
	return c.readOnlyFunc(nodeName)
}

// patchResource patches the resource
func (c *NodeController) patchResource(ctx context.Context, node *corev1.Node, patch []byte) (*corev1.Node, error) {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)

	result, err := c.typedClient.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch node",
				"err", err,
			)
			return nil, nil
		}
		return nil, err
	}
	logger.Info("Patch node")
	return result, nil
}

func (c *NodeController) computePatch(node *corev1.Node, tpl string) ([]byte, error) {
	patch, err := c.renderer.ToJSON(tpl, node)
	if err != nil {
		return nil, err
	}

	original, err := json.Marshal(node.Status)
	if err != nil {
		return nil, err
	}

	sum, err := strategicpatch.StrategicMergePatch(original, patch, node.Status)
	if err != nil {
		return nil, err
	}

	nodeStatus := corev1.NodeStatus{}
	err = json.Unmarshal(sum, &nodeStatus)
	if err != nil {
		return nil, err
	}

	dist, err := json.Marshal(nodeStatus)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(original, dist) {
		return nil, nil
	}

	return json.Marshal(map[string]json.RawMessage{
		"status": patch,
	})
}

// putNodeInfo puts node info
func (c *NodeController) putNodeInfo(node *corev1.Node) {
	nodeIPs := getNodeHostIPs(node)
	hostIps := slices.Map(nodeIPs, func(ip net.IP) string {
		return ip.String()
	})

	podCIDRs := node.Spec.PodCIDRs
	if len(podCIDRs) == 0 && node.Spec.PodCIDR != "" {
		podCIDRs = []string{node.Spec.PodCIDR}
	}

	nodeInfo := &NodeInfo{
		Node:     node,
		HostIPs:  hostIps,
		PodCIDRs: podCIDRs,
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: nodeKind.Version,
				Kind:       nodeKind.Kind,
				Name:       node.Name,
				UID:        node.UID,
			},
		},
	}
	c.nodesSets.Store(node.Name, nodeInfo)
}

// deleteNodeInfo deletes node info
func (c *NodeController) deleteNodeInfo(node *corev1.Node) {
	c.nodesSets.Delete(node.Name)
}

// Manage manages the node
func (c *NodeController) Manage(nodeName string) {
	c.triggerPreprocessChan <- nodeName
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

// Has returns true if the node is existed
func (c *NodeController) Has(nodeName string) bool {
	_, has := c.nodesSets.Load(nodeName)
	return has
}

// Size returns the number of nodes
func (c *NodeController) Size() int {
	return c.nodesSets.Size()
}

// Get returns Has bool and node info
func (c *NodeController) Get(nodeName string) (*NodeInfo, bool) {
	nodeInfo, has := c.nodesSets.Load(nodeName)
	if has {
		return nodeInfo, has
	}
	return nil, has
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
