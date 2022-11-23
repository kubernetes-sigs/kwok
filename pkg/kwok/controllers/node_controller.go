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
	"sort"
	"text/template"
	"time"

	"sigs.k8s.io/kwok/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/pager"
)

// NodeController is a fake nodes implementation that can be used to test
type NodeController struct {
	clientSet                             kubernetes.Interface
	nodeIP                                string
	disregardStatusWithAnnotationSelector labels.Selector
	disregardStatusWithLabelSelector      labels.Selector
	manageNodesWithAnnotationSelector     string
	manageNodesWithLabelSelector          string
	nodeSelectorFunc                      func(node *corev1.Node) bool
	lockPodsOnNodeFunc                    func(ctx context.Context, nodeName string) error
	nodesSets                             *stringSets
	nodeHeartbeatTemplate                 string
	nodeStatusTemplate                    string
	funcMap                               template.FuncMap
	logger                                logger.Logger
	nodeHeartbeatInterval                 time.Duration
	nodeHeartbeatParallelism              int
	lockNodeParallelism                   int
	nodeChan                              chan string
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
	NodeStatusTemplate                    string
	NodeHeartbeatTemplate                 string
	Logger                                logger.Logger
	NodeHeartbeatInterval                 time.Duration
	NodeHeartbeatParallelism              int
	LockNodeParallelism                   int
	FuncMap                               template.FuncMap
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

	log := conf.Logger
	if log == nil {
		log = logger.Noop
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
		nodesSets:                             newStringSets(),
		logger:                                log,
		nodeHeartbeatTemplate:                 conf.NodeHeartbeatTemplate,
		nodeStatusTemplate:                    conf.NodeStatusTemplate + "\n" + conf.NodeHeartbeatTemplate,
		nodeHeartbeatInterval:                 conf.NodeHeartbeatInterval,
		nodeHeartbeatParallelism:              conf.NodeHeartbeatParallelism,
		lockNodeParallelism:                   conf.LockNodeParallelism,
		nodeChan:                              make(chan string),
	}
	n.funcMap = template.FuncMap{
		"NodeIP": func() string {
			return n.nodeIP
		},
	}
	for k, v := range conf.FuncMap {
		n.funcMap[k] = v
	}

	return n, nil
}

// Start starts the fake nodes controller
// if nodeSelectorFunc is not nil, it will use it to determine if the node should be managed
func (c *NodeController) Start(ctx context.Context) error {
	go c.KeepNodeHeartbeat(ctx)

	go c.LockNodes(ctx, c.nodeChan)

	opt := metav1.ListOptions{
		LabelSelector: c.manageNodesWithLabelSelector,
	}
	err := c.WatchNodes(ctx, c.nodeChan, opt)
	if err != nil {
		return fmt.Errorf("failed watch node: %w", err)
	}

	go func() {
		err = c.ListNodes(ctx, c.nodeChan, opt)
		if err != nil {
			c.logger.Printf("failed list node: %s", err)
		}
	}()

	return nil
}

func (c *NodeController) heartbeatNode(ctx context.Context, nodeName string) error {
	var node corev1.Node
	node.Name = nodeName
	patch, err := c.configureHeartbeatNode(&node)
	if err != nil {
		return err
	}
	_, err = c.clientSet.CoreV1().Nodes().PatchStatus(ctx, node.Name, patch)
	if err != nil {
		return err
	}
	return nil
}

func (c *NodeController) allHeartbeatNode(ctx context.Context, nodes []string, tasks *parallelTasks) {
	for _, node := range nodes {
		localNode := node
		tasks.Add(func() {
			err := c.heartbeatNode(ctx, localNode)
			if err != nil {
				c.logger.Printf("Failed to heartbeat node %s: %s", localNode, err)
			}
		})
	}
}

// KeepNodeHeartbeat keep node heartbeat
func (c *NodeController) KeepNodeHeartbeat(ctx context.Context) {
	th := time.NewTimer(c.nodeHeartbeatInterval)
	tasks := newParallelTasks(c.nodeHeartbeatParallelism)
	var heartbeatStartTime time.Time
	var nodes []string
loop:
	for {
		select {
		case <-th.C:
			nodes = nodes[:0]
			c.nodesSets.Foreach(func(node string) {
				nodes = append(nodes, node)
			})
			sort.Strings(nodes)
			heartbeatStartTime = time.Now()
			c.allHeartbeatNode(ctx, nodes, tasks)
			tasks.Wait()
			c.logger.Printf("Heartbeat %d nodes took %s", len(nodes), time.Since(heartbeatStartTime))
			th.Reset(c.nodeHeartbeatInterval)
		case <-ctx.Done():
			c.logger.Printf("Stop keep nodes heartbeat")
			break loop
		}
	}
	tasks.Wait()
}

func (c *NodeController) needHeartbeat(node *corev1.Node) bool {
	return c.nodeSelectorFunc(node)
}

func (c *NodeController) needLockNode(node *corev1.Node) bool {
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
func (c *NodeController) WatchNodes(ctx context.Context, ch chan<- string, opt metav1.ListOptions) error {
	// Watch nodes in the cluster
	watcher, err := c.clientSet.CoreV1().Nodes().Watch(ctx, opt)
	if err != nil {
		return err
	}
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

						c.logger.Printf("Failed to watch nodes: %s", err)
						select {
						case <-ctx.Done():
							break loop
						case <-time.After(time.Second * 5):
						}
					}
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					node := event.Object.(*corev1.Node)
					if c.needHeartbeat(node) {
						c.nodesSets.Put(node.Name)
						if c.needLockNode(node) {
							ch <- node.Name
						}
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
		c.logger.Printf("Stop watch nodes")
	}()
	return nil
}

// ListNodes list nodes put into the channel
func (c *NodeController) ListNodes(ctx context.Context, ch chan<- string, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientSet.CoreV1().Nodes().List(ctx, opts)
	})
	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		node := obj.(*corev1.Node)
		if c.needHeartbeat(node) {
			c.nodesSets.Put(node.Name)
			if c.needLockNode(node) {
				ch <- node.Name
			}
		}
		return nil
	})
}

// LockNodes locks a nodes from the channel
// if they don't exist we create them and then manage them
// if they exist we manage them
func (c *NodeController) LockNodes(ctx context.Context, nodes <-chan string) {
	tasks := newParallelTasks(c.lockNodeParallelism)
	for node := range nodes {
		if node == "" {
			continue
		}
		localNode := node
		tasks.Add(func() {
			err := c.LockNode(ctx, localNode)
			if err != nil {
				c.logger.Printf("Failed to lock node %s: %s", localNode, err)
				return
			}
			if c.lockPodsOnNodeFunc != nil {
				err = c.lockPodsOnNodeFunc(ctx, localNode)
				if err != nil {
					c.logger.Printf("Failed to lock pods on node %s: %s", localNode, err)
					return
				}
			}
		})
	}
	tasks.Wait()
}

// LockNode locks a given node
func (c *NodeController) LockNode(ctx context.Context, nodeName string) error {
	node, err := c.clientSet.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	patch, err := c.configureNode(node)
	if err != nil {
		return err
	}
	if patch == nil {
		return nil
	}
	_, err = c.clientSet.CoreV1().Nodes().PatchStatus(ctx, node.Name, patch)
	if err != nil {
		return err
	}
	c.logger.Printf("Lock node %s", nodeName)
	return nil
}

func (c *NodeController) configureNode(node *corev1.Node) ([]byte, error) {
	patch, err := toTemplateJson(c.nodeStatusTemplate, node, c.funcMap)
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
	nodeStatus.Conditions = node.Status.Conditions

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

func (c *NodeController) configureHeartbeatNode(node *corev1.Node) ([]byte, error) {
	patch, err := toTemplateJson(c.nodeHeartbeatTemplate, node, c.funcMap)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{
		"status": patch,
	})
}

func (c *NodeController) Has(nodeName string) bool {
	return c.nodesSets.Has(nodeName)
}

func (c *NodeController) Size() int {
	return c.nodesSets.Size()
}
