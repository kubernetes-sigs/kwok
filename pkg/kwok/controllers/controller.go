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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

var (
	startTime = time.Now().Format(time.RFC3339Nano)

	defaultFuncMap = gotpl.FuncMap{
		"Quote": func(s any) string {
			data, err := json.Marshal(s)
			if err != nil {
				return strconv.Quote(fmt.Sprint(s))
			}
			if len(data) == 0 {
				return `""`
			}
			if data[0] == '"' {
				return string(data)
			}
			return strconv.Quote(string(data))
		},
		"Now": func() string {
			return time.Now().Format(time.RFC3339Nano)
		},
		"StartTime": func() string {
			return startTime
		},
		"YAML": func(s interface{}, indent ...int) (string, error) {
			d, err := yaml.Marshal(s)
			if err != nil {
				return "", err
			}

			data := string(d)
			if len(indent) == 1 && indent[0] > 0 {
				pad := strings.Repeat(" ", indent[0]*2)
				data = strings.ReplaceAll("\n"+data, "\n", "\n"+pad)
			}
			return data, nil
		},
		"Version": func() string {
			return consts.Version
		},
	}

	nodeKind = corev1.SchemeGroupVersion.WithKind("Node")
)

// Controller is a fake kubelet implementation that can be used to test
type Controller struct {
	conf        Config
	nodes       *NodeController
	pods        *PodController
	nodeLeases  *NodeLeaseController
	broadcaster record.EventBroadcaster
	typedClient kubernetes.Interface

	nodeCacheGetter informer.Getter[*corev1.Node]
	podCacheGetter  informer.Getter[*corev1.Pod]
}

// Config is the configuration for the controller
type Config struct {
	Clock                                 clock.Clock
	EnableCNI                             bool
	TypedClient                           kubernetes.Interface
	TypedKwokClient                       versioned.Interface
	ManageSingleNode                      string
	ManageAllNodes                        bool
	ManageNodesWithAnnotationSelector     string
	ManageNodesWithLabelSelector          string
	DisregardStatusWithAnnotationSelector string
	DisregardStatusWithLabelSelector      string
	CIDR                                  string
	NodeIP                                string
	NodeName                              string
	NodePort                              int
	PodStages                             []*internalversion.Stage
	NodeStages                            []*internalversion.Stage
	PodPlayStageParallelism               uint
	NodePlayStageParallelism              uint
	NodeLeaseDurationSeconds              uint
	NodeLeaseParallelism                  uint
	ID                                    string
	EnableMetrics                         bool
	EnablePodCache                        bool
}

func (c Config) validate() error {
	switch {
	case c.ManageSingleNode != "":
		if c.ManageAllNodes {
			return fmt.Errorf("manage-single-node is conflicted with manage-all-nodes")
		}
		if c.ManageNodesWithAnnotationSelector != "" || c.ManageNodesWithLabelSelector != "" {
			return fmt.Errorf("manage-single-node is conflicted with manage-nodes-with-annotation-selector or manage-nodes-with-label-selector")
		}
	case c.ManageAllNodes:
		if c.ManageNodesWithAnnotationSelector != "" || c.ManageNodesWithLabelSelector != "" {
			return fmt.Errorf("manage-all-nodes is conflicted with manage-nodes-with-annotation-selector or manage-nodes-with-label-selector")
		}
	case c.ManageNodesWithAnnotationSelector != "" || c.ManageNodesWithLabelSelector != "":
	default:
		return fmt.Errorf("no nodes are managed")
	}
	return nil
}

// NewController creates a new fake kubelet controller
func NewController(conf Config) (*Controller, error) {
	err := conf.validate()
	if err != nil {
		return nil, err
	}

	n := &Controller{
		conf:        conf,
		broadcaster: record.NewBroadcaster(),
		typedClient: conf.TypedClient,
	}

	return n, nil
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	if c.pods != nil || c.nodes != nil || c.nodeLeases != nil {
		return fmt.Errorf("controller already started")
	}

	conf := c.conf

	recorder := c.broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "kwok_controller"})

	var (
		err                   error
		nodeLeases            *NodeLeaseController
		nodeLeasesChan        chan informer.Event[*coordinationv1.Lease]
		onLeaseNodeManageFunc func(nodeName string)
		onNodeManagedFunc     func(nodeName string)
		readOnlyFunc          func(nodeName string) bool

		manageNodesWithLabelSelector      string
		manageNodesWithAnnotationSelector string
		manageNodesWithFieldSelector      string
		manageNodeLeasesWithFieldSelector string
		managePodsWithFieldSelector       string
	)

	switch {
	case conf.ManageSingleNode != "":
		managePodsWithFieldSelector = fields.OneTermEqualSelector("spec.nodeName", conf.ManageSingleNode).String()
		manageNodesWithFieldSelector = fields.OneTermEqualSelector("metadata.name", conf.ManageSingleNode).String()
		manageNodeLeasesWithFieldSelector = fields.OneTermEqualSelector("metadata.name", conf.ManageSingleNode).String()
	case conf.ManageAllNodes:
		managePodsWithFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
	case conf.ManageNodesWithLabelSelector != "" || conf.ManageNodesWithAnnotationSelector != "":
		manageNodesWithLabelSelector = conf.ManageNodesWithLabelSelector
		manageNodesWithAnnotationSelector = conf.ManageNodesWithAnnotationSelector
		managePodsWithFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
	}

	nodeChan := make(chan informer.Event[*corev1.Node], 1)
	nodesCli := conf.TypedClient.CoreV1().Nodes()
	nodesInformer := informer.NewInformer[*corev1.Node, *corev1.NodeList](nodesCli)
	nodesCache, err := nodesInformer.WatchWithCache(ctx, informer.Option{
		LabelSelector:      manageNodesWithLabelSelector,
		AnnotationSelector: manageNodesWithAnnotationSelector,
		FieldSelector:      manageNodesWithFieldSelector,
	}, nodeChan)
	if err != nil {
		return fmt.Errorf("failed to watch nodes: %w", err)
	}

	podsChan := make(chan informer.Event[*corev1.Pod], 1)
	podsCli := conf.TypedClient.CoreV1().Pods(corev1.NamespaceAll)
	podsInformer := informer.NewInformer[*corev1.Pod, *corev1.PodList](podsCli)

	podWatchOption := informer.Option{
		FieldSelector: managePodsWithFieldSelector,
	}

	var podsCache informer.Getter[*corev1.Pod]
	if conf.EnablePodCache {
		podsCache, err = podsInformer.WatchWithCache(ctx, podWatchOption, podsChan)
	} else {
		err = podsInformer.Watch(ctx, podWatchOption, podsChan)
	}
	if err != nil {
		return fmt.Errorf("failed to watch pods: %w", err)
	}

	if conf.NodeLeaseDurationSeconds != 0 {
		nodeLeasesChan = make(chan informer.Event[*coordinationv1.Lease], 1)
		nodeLeasesCli := conf.TypedClient.CoordinationV1().Leases(corev1.NamespaceNodeLease)
		nodeLeasesInformer := informer.NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](nodeLeasesCli)
		err = nodeLeasesInformer.Watch(ctx, informer.Option{
			FieldSelector: manageNodeLeasesWithFieldSelector,
		}, nodeLeasesChan)
		if err != nil {
			return fmt.Errorf("failed to watch node leases: %w", err)
		}

		leaseDuration := time.Duration(conf.NodeLeaseDurationSeconds) * time.Second
		// https://github.com/kubernetes/kubernetes/blob/02f4d643eae2e225591702e1bbf432efea453a26/pkg/kubelet/kubelet.go#L199-L200
		renewInterval := leaseDuration / 4
		// https://github.com/kubernetes/component-helpers/blob/d17b6f1e84500ee7062a26f5327dc73cb3e9374a/apimachinery/lease/controller.go#L100
		renewIntervalJitter := 0.04
		nodeLeases, err = NewNodeLeaseController(NodeLeaseControllerConfig{
			Clock:                conf.Clock,
			TypedClient:          conf.TypedClient,
			NodeCacheGetter:      nodesCache,
			LeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
			LeaseParallelism:     conf.NodeLeaseParallelism,
			RenewInterval:        renewInterval,
			RenewIntervalJitter:  renewIntervalJitter,
			MutateLeaseFunc: setNodeOwnerFunc(func(nodeName string) []metav1.OwnerReference {
				node, ok := nodesCache.Get(nodeName)
				if !ok {
					return nil
				}
				ownerReferences := []metav1.OwnerReference{
					{
						APIVersion: nodeKind.Version,
						Kind:       nodeKind.Kind,
						Name:       node.Name,
						UID:        node.UID,
					},
				}
				return ownerReferences
			}),
			HolderIdentity: conf.ID,
			OnNodeManagedFunc: func(nodeName string) {
				onLeaseNodeManageFunc(nodeName)
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create node leases controller: %w", err)
		}

		// Not holding the lease means the node is not managed
		readOnlyFunc = func(nodeName string) bool {
			return !nodeLeases.Held(nodeName)
		}
	}

	logger := log.FromContext(ctx)

	var nodeLifecycleGetter resources.Getter[Lifecycle]
	var podLifecycleGetter resources.Getter[Lifecycle]

	if len(conf.PodStages) == 0 && len(conf.NodeStages) == 0 {
		getter := resources.NewDynamicGetter[
			[]*internalversion.Stage,
			*v1alpha1.Stage,
			*v1alpha1.StageList,
		](
			conf.TypedKwokClient.KwokV1alpha1().Stages(),
			func(objs []*v1alpha1.Stage) []*internalversion.Stage {
				return slices.FilterAndMap(objs, func(obj *v1alpha1.Stage) (*internalversion.Stage, bool) {
					r, err := internalversion.ConvertToInternalStage(obj)
					if err != nil {
						logger.Error("failed to convert to internal stage", err, "obj", obj)
						return nil, false
					}
					return r, true
				})
			},
		)

		nodeLifecycleGetter = resources.NewFilter[Lifecycle, []*internalversion.Stage](getter, func(stages []*internalversion.Stage) Lifecycle {
			lifecycle := slices.FilterAndMap(stages, func(stage *internalversion.Stage) (*LifecycleStage, bool) {
				if stage.Spec.ResourceRef.Kind != "Node" {
					return nil, false
				}

				lifecycleStage, err := NewLifecycleStage(stage)
				if err != nil {
					logger.Error("failed to create node lifecycle stage", err, "stage", stage)
					return nil, false
				}
				return lifecycleStage, true
			})
			return lifecycle
		})

		podLifecycleGetter = resources.NewFilter[Lifecycle, []*internalversion.Stage](getter, func(stages []*internalversion.Stage) Lifecycle {
			lifecycle := slices.FilterAndMap(stages, func(stage *internalversion.Stage) (*LifecycleStage, bool) {
				if stage.Spec.ResourceRef.Kind != "Pod" {
					return nil, false
				}

				lifecycleStage, err := NewLifecycleStage(stage)
				if err != nil {
					logger.Error("failed to create node lifecycle stage", err, "stage", stage)
					return nil, false
				}
				return lifecycleStage, true
			})
			return lifecycle
		})

		err := getter.Start(ctx)
		if err != nil {
			return err
		}
	} else {
		lifecycle, err := NewLifecycle(conf.PodStages)
		if err != nil {
			return fmt.Errorf("failed to create pod lifecycle: %w", err)
		}
		podLifecycleGetter = resources.NewStaticGetter(lifecycle)

		lifecycle, err = NewLifecycle(conf.NodeStages)
		if err != nil {
			return fmt.Errorf("failed to create node lifecycle: %w", err)
		}
		nodeLifecycleGetter = resources.NewStaticGetter(lifecycle)
	}

	nodes, err := NewNodeController(NodeControllerConfig{
		Clock:                                 conf.Clock,
		TypedClient:                           conf.TypedClient,
		NodeCacheGetter:                       nodesCache,
		NodeIP:                                conf.NodeIP,
		NodeName:                              conf.NodeName,
		NodePort:                              conf.NodePort,
		DisregardStatusWithAnnotationSelector: conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      conf.DisregardStatusWithLabelSelector,
		OnNodeManagedFunc: func(nodeName string) {
			onNodeManagedFunc(nodeName)
		},
		Lifecycle:            nodeLifecycleGetter,
		PlayStageParallelism: conf.NodePlayStageParallelism,
		FuncMap:              defaultFuncMap,
		Recorder:             recorder,
		ReadOnlyFunc:         readOnlyFunc,
		EnableMetrics:        conf.EnableMetrics,
	})
	if err != nil {
		return fmt.Errorf("failed to create nodes controller: %w", err)
	}

	pods, err := NewPodController(PodControllerConfig{
		Clock:                                 conf.Clock,
		EnableCNI:                             conf.EnableCNI,
		TypedClient:                           conf.TypedClient,
		NodeCacheGetter:                       nodesCache,
		NodeIP:                                conf.NodeIP,
		CIDR:                                  conf.CIDR,
		DisregardStatusWithAnnotationSelector: conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      conf.DisregardStatusWithLabelSelector,
		Lifecycle:                             podLifecycleGetter,
		PlayStageParallelism:                  conf.PodPlayStageParallelism,
		NodeGetFunc:                           nodes.Get,
		FuncMap:                               defaultFuncMap,
		Recorder:                              recorder,
		ReadOnlyFunc:                          readOnlyFunc,
		EnableMetrics:                         conf.EnableMetrics,
	})
	if err != nil {
		return fmt.Errorf("failed to create pods controller: %w", err)
	}

	podOnNodeManageQueue := queue.NewQueue[string]()
	if nodeLeases != nil {
		nodeManageQueue := queue.NewQueue[string]()
		onLeaseNodeManageFunc = func(nodeName string) {
			nodeManageQueue.Add(nodeName)
			podOnNodeManageQueue.Add(nodeName)
		}
		onNodeManagedFunc = func(nodeName string) {
			// Try to hold the lease
			nodeLeases.TryHold(nodeName)
		}

		go func() {
			for {
				nodeName := nodeManageQueue.GetOrWait()
				node, ok := nodesCache.Get(nodeName)
				if !ok {
					logger.Warn("node not found in cache", "node", nodeName)
					err := nodesInformer.Sync(ctx, informer.Option{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", nodeName).String(),
					}, nodeChan)
					if err != nil {
						logger.Error("failed to update node", err, "node", nodeName)
					}
					continue
				}
				nodeChan <- informer.Event[*corev1.Node]{
					Type:   informer.Sync,
					Object: node,
				}
			}
		}()
	} else {
		onNodeManagedFunc = func(nodeName string) {
			podOnNodeManageQueue.Add(nodeName)
		}
	}

	go func() {
		for {
			nodeName := podOnNodeManageQueue.GetOrWait()
			err = podsInformer.Sync(ctx, informer.Option{
				FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
			}, podsChan)
			if err != nil {
				logger.Error("failed to update pods on node", err, "node", nodeName)
			}
		}
	}()

	c.broadcaster.StartRecordingToSink(&clientcorev1.EventSinkImpl{Interface: c.typedClient.CoreV1().Events("")})
	if nodeLeases != nil {
		err := nodeLeases.Start(ctx, nodeLeasesChan)
		if err != nil {
			return fmt.Errorf("failed to start node leases controller: %w", err)
		}
	}
	err = pods.Start(ctx, podsChan)
	if err != nil {
		return fmt.Errorf("failed to start pods controller: %w", err)
	}
	err = nodes.Start(ctx, nodeChan)
	if err != nil {
		return fmt.Errorf("failed to start nodes controller: %w", err)
	}

	c.pods = pods
	c.nodes = nodes
	c.nodeLeases = nodeLeases
	c.nodeCacheGetter = nodesCache
	c.podCacheGetter = podsCache
	return nil
}

// ListNodes returns all nodes
func (c *Controller) ListNodes() []string {
	return c.nodes.List()
}

// ListPods returns all pods on the given node
func (c *Controller) ListPods(nodeName string) ([]log.ObjectRef, bool) {
	return c.pods.List(nodeName)
}

// GetPodCache returns the pod cache
func (c *Controller) GetPodCache() informer.Getter[*corev1.Pod] {
	return c.podCacheGetter
}

// GetNodeCache returns the node cache
func (c *Controller) GetNodeCache() informer.Getter[*corev1.Node] {
	return c.nodeCacheGetter
}

// StartedContainersTotal returns the total number of containers started
func (c *Controller) StartedContainersTotal(nodeName string) int64 {
	nodeInfo, ok := c.nodes.Get(nodeName)
	if !ok {
		return 0
	}
	return nodeInfo.StartedContainer.Load()
}

// Identity returns a unique identifier for this controller
func Identity() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("unable to get hostname: %w", err)
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	return hostname + "_" + string(uuid.NewUUID()), nil
}
