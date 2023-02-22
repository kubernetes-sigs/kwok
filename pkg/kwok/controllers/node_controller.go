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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/pager"
	"k8s.io/client-go/tools/record"
	netutils "k8s.io/utils/net"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/expression"
)

// NodeController is a fake nodes implementation that can be used to test
type NodeController struct {
	clientSet                             kubernetes.Interface
	nodeIP                                string
	nodeName                              string
	nodePort                              int
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	manageNodesWithAnnotationSelector     string
	manageNodesWithLabelSelector          string
	nodeSelectorFunc                      func(node *corev1.Node) bool
	lockPodsOnNodeFunc                    func(ctx context.Context, nodeName string) error
	nodesSets                             *stringSets
	renderer                              *renderer
	nodeChan                              chan *corev1.Node
	parallelTasks                         *parallelTasks
	lifecycle                             Lifecycle
	cronjob                               *cron.Cron
	delayJobsCancels                      sync.Map
	recorder                              record.EventRecorder
}

// NodeControllerConfig is the configuration for the NodeController
type NodeControllerConfig struct {
	ClientSet                             kubernetes.Interface
	NodeSelectorFunc                      func(node *corev1.Node) bool
	LockPodsOnNodeFunc                    func(ctx context.Context, nodeName string) error
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	ManageNodesWithAnnotationSelector     string
	ManageNodesWithLabelSelector          string
	NodeIP                                string
	NodeName                              string
	NodePort                              int
	Stages                                []*internalversion.Stage
	LockNodeParallelism                   int
	FuncMap                               template.FuncMap
	Recorder                              record.EventRecorder
}

// NodeInfo is the collection of necessary node information
type NodeInfo struct {
	HostIPs []string
}

// NewNodeController creates a new fake nodes controller
func NewNodeController(conf NodeControllerConfig) (*NodeController, error) {
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

	n := &NodeController{
		clientSet:                             conf.ClientSet,
		nodeSelectorFunc:                      conf.NodeSelectorFunc,
		disregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
		disregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
		manageNodesWithAnnotationSelector:     conf.ManageNodesWithAnnotationSelector,
		manageNodesWithLabelSelector:          conf.ManageNodesWithLabelSelector,
		lockPodsOnNodeFunc:                    conf.LockPodsOnNodeFunc,
		nodeIP:                                conf.NodeIP,
		nodeName:                              conf.NodeName,
		nodePort:                              conf.NodePort,
		nodesSets:                             newStringSets(),
		cronjob:                               cron.NewCron(),
		lifecycle:                             lifecycles,
		parallelTasks:                         newParallelTasks(conf.LockNodeParallelism),
		nodeChan:                              make(chan *corev1.Node),
		recorder:                              conf.Recorder,
	}
	funcMap := template.FuncMap{
		"NodeIP": func() string {
			return n.nodeIP
		},
		"NodeName": func() string {
			return n.nodeName
		},
		"NodePort": func() int {
			return n.nodePort
		},
	}
	for k, v := range conf.FuncMap {
		funcMap[k] = v
	}
	n.renderer = newRenderer(funcMap)
	return n, nil
}

// Start starts the fake nodes controller
// if nodeSelectorFunc is not nil, it will use it to determine if the node should be managed
func (c *NodeController) Start(ctx context.Context) error {
	go c.LockNodes(ctx, c.nodeChan)

	opt := metav1.ListOptions{
		LabelSelector: c.manageNodesWithLabelSelector,
	}
	err := c.WatchNodes(ctx, c.nodeChan, opt)
	if err != nil {
		return fmt.Errorf("failed watch node: %w", err)
	}

	logger := log.FromContext(ctx)
	go func() {
		err = c.ListNodes(ctx, c.nodeChan, opt)
		if err != nil {
			logger.Error("Failed list node", err)
		}
	}()

	return nil
}

func (c *NodeController) needLockNode(node *corev1.Node) bool {
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

// WatchNodes watch nodes put into the channel
func (c *NodeController) WatchNodes(ctx context.Context, ch chan<- *corev1.Node, opt metav1.ListOptions) error {
	// Watch nodes in the cluster
	watcher, err := c.clientSet.CoreV1().Nodes().Watch(ctx, opt)
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
						watcher, err := c.clientSet.CoreV1().Nodes().Watch(ctx, opt)
						if err == nil {
							rc = watcher.ResultChan()
							continue loop
						}

						logger.Error("Failed to watch nodes", err)
						select {
						case <-ctx.Done():
							break loop
						case <-time.After(time.Second * 5):
						}
					}
				}
				switch event.Type {
				case watch.Added:
					node := event.Object.(*corev1.Node)
					if c.needLockNode(node) {
						c.putNodeInfo(node)
						ch <- node
						if c.lockPodsOnNodeFunc != nil {
							err = c.lockPodsOnNodeFunc(ctx, node.Name)
							if err != nil {
								logger.Error("Failed to lock pods on node", err,
									"node", node.Name,
								)
								return
							}
						}
					}
				case watch.Modified:
					node := event.Object.(*corev1.Node)
					if c.needLockNode(node) {
						c.putNodeInfo(node)
						ch <- node
					}
				case watch.Deleted:
					node := event.Object.(*corev1.Node)
					if c.nodesSets.Has(node.Name) {
						c.nodesSets.Delete(node.Name)
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

// ListNodes list nodes put into the channel
func (c *NodeController) ListNodes(ctx context.Context, ch chan<- *corev1.Node, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientSet.CoreV1().Nodes().List(ctx, opts)
	})
	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		node := obj.(*corev1.Node)
		if c.needLockNode(node) {
			c.putNodeInfo(node)
			ch <- node
		}
		return nil
	})
}

// LockNodes locks a nodes from the channel
func (c *NodeController) LockNodes(ctx context.Context, nodes <-chan *corev1.Node) {
	logger := log.FromContext(ctx)
	for node := range nodes {
		localNode := node
		c.parallelTasks.Add(func() {
			err := c.LockNode(ctx, localNode)
			if err != nil {
				logger.Error("Failed to lock node", err,
					"node", localNode.Name,
				)
				return
			}
		})
	}
	c.parallelTasks.Wait()
}

// FinalizersModify modify finalizers of node
func (c *NodeController) FinalizersModify(ctx context.Context, node *corev1.Node, finalizers *internalversion.StageFinalizers) error {
	ops := finalizersModify(node.Finalizers, finalizers)
	if len(ops) == 0 {
		return nil
	}
	data, err := json.Marshal(ops)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)
	_, err = c.clientSet.CoreV1().Nodes().Patch(ctx, node.Name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch node finalizers",
				"err", err,
			)
			return nil
		}
		return err
	}
	logger.Info("Patch node finalizers")
	return nil
}

// DeleteNode deletes a node
func (c *NodeController) DeleteNode(ctx context.Context, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)
	err := c.clientSet.CoreV1().Nodes().Delete(ctx, node.Name, deleteOpt)
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

// LockNode locks a given node
func (c *NodeController) LockNode(ctx context.Context, node *corev1.Node) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)
	data, err := expression.ToJSONStandard(node)
	if err != nil {
		return err
	}
	key := node.Name

	_, ok := c.delayJobsCancels.Load(key)
	if ok {
		return nil
	}

	stage, err := c.lifecycle.Match(node.Labels, node.Annotations, data)
	if err != nil {
		return fmt.Errorf("stage match: %w", err)
	}
	if stage == nil {
		logger.Debug("Skip node",
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
				Kind:      "Node",
				UID:       node.UID,
				Name:      node.Name,
				Namespace: "",
			}, next.Event.Type, next.Event.Reason, next.Event.Message)
		}
		if next.Finalizers != nil {
			err = c.FinalizersModify(ctx, node, next.Finalizers)
			if err != nil {
				logger.Error("Failed to finalizers of node", err)
			}
		}
		if next.Delete {
			err = c.DeleteNode(ctx, node)
			if err != nil {
				logger.Error("Failed to delete node", err)
			}
		} else if next.StatusTemplate != "" {
			patch, err := c.computePatchNode(node, next.StatusTemplate)
			if err != nil {
				logger.Error("Failed to configure node", err)
				return
			}
			if patch == nil {
				logger.Debug("Skip node",
					"reason", "do not need to modify",
				)
			} else {
				err = c.lockNode(ctx, node, patch)
				if err != nil {
					logger.Error("Failed to lock node", err)
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

func (c *NodeController) lockNode(ctx context.Context, node *corev1.Node, patch []byte) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"node", node.Name,
	)
	_, err := c.clientSet.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "status")
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Warn("Patch node",
				"err", err,
			)
			return nil
		}
		return err
	}
	logger.Info("Lock node")
	return nil
}

func (c *NodeController) computePatchNode(node *corev1.Node, tpl string) ([]byte, error) {
	patch, err := c.renderer.renderToJSON(tpl, node)
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

// putNodeInfo puts node info (HostIPs)
func (c *NodeController) putNodeInfo(node *corev1.Node) {
	nodeIPs := []string{}
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
		// ipv4 then ipv6
		for _, ip := range allIPs {
			if netutils.IsIPv4(ip) {
				nodeIPs = append(nodeIPs, ip.To4().String())
				break
			}
		}
		for _, ip := range allIPs {
			if netutils.IsIPv6(ip) {
				nodeIPs = append(nodeIPs, ip.To16().String())
				break
			}
		}
	}
	nodeInfo := NodeInfo{
		HostIPs: nodeIPs,
	}
	c.nodesSets.Put(node.Name, nodeInfo)
}

// Has returns true if the node is existed
func (c *NodeController) Has(nodeName string) bool {
	return c.nodesSets.Has(nodeName)
}

// Size returns the number of nodes
func (c *NodeController) Size() int {
	return c.nodesSets.Size()
}

// Get returns Has bool and corev1.Node if the node is existed
func (c *NodeController) Get(nodeName string) (NodeInfo, bool) {
	value, has := c.nodesSets.Get(nodeName)
	if nodeInfo, converted := value.(NodeInfo); has && converted {
		return nodeInfo, has
	}
	return NodeInfo{}, has
}
