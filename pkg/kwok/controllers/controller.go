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
	"maps"
	"os"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

var (
	nodeKind = corev1.SchemeGroupVersion.WithKind("Node")
)

// Controller is a fake kubelet implementation that can be used to test
type Controller struct {
	conf Config
	env  *cel.Environment

	stagesManager *StagesManager

	nodes       *NodeController
	pods        *PodController
	nodeLeases  *NodeLeaseController
	broadcaster record.EventBroadcaster
	recorder    record.EventRecorder

	nodeCacheGetter      informer.Getter[*corev1.Node]
	podCacheGetter       informer.Getter[*corev1.Pod]
	nodeLeaseCacheGetter informer.Getter[*coordinationv1.Lease]

	onNodeManagedFunc   func(nodeName string)
	onNodeUnmanagedFunc func(nodeName string)
	readOnlyFunc        func(nodeName string) bool

	manageNodesWithLabelSelector      string
	manageNodesWithAnnotationSelector string
	manageNodesWithFieldSelector      string
	manageNodeLeasesWithFieldSelector string
	managePodsWithFieldSelector       string

	nodesChan chan informer.Event[*corev1.Node]
	podsChan  chan informer.Event[*corev1.Pod]

	nodeLeasesInformer *informer.Informer[*coordinationv1.Lease, *coordinationv1.LeaseList]
	nodesInformer      *informer.Informer[*corev1.Node, *corev1.NodeList]
	podsInformer       *informer.Informer[*corev1.Pod, *corev1.PodList]

	patchMeta *patch.PatchMetaFromOpenAPI3

	stageGetter resources.DynamicGetter[[]*internalversion.Stage]

	podOnNodeManageQueue queue.Queue[string]
	nodeManageQueue      queue.Queue[string]
}

// Config is the configuration for the controller
type Config struct {
	Clock                                 clock.Clock
	EnableCNI                             bool
	DynamicClient                         dynamic.Interface
	RESTClient                            rest.Interface
	ImpersonatingDynamicClient            client.DynamicClientImpersonator
	RESTMapper                            meta.RESTMapper
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
	LocalStages                           map[internalversion.StageResourceRef][]*internalversion.Stage
	PodPlayStageParallelism               uint
	NodePlayStageParallelism              uint
	NodeLeaseDurationSeconds              uint
	NodeLeaseParallelism                  uint
	PodsOnNodeSyncParallelism             uint
	EnablePodsOnNodeSyncListPager         bool
	EnablePodsOnNodeSyncStreamWatch       bool
	ID                                    string
	EnableMetrics                         bool
	EnablePodCache                        bool
	FuncMap                               gotpl.FuncMap
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

	types := slices.Clone(cel.DefaultTypes)
	conversions := slices.Clone(cel.DefaultConversions)
	funcs := maps.Clone(cel.DefaultFuncs)
	methods := maps.Clone(cel.FuncsToMethods(cel.DefaultFuncs))
	vars := map[string]any{
		"self": map[any]any{},
	}
	env, err := cel.NewEnvironment(cel.EnvironmentConfig{
		Types:       types,
		Conversions: conversions,
		Methods:     methods,
		Funcs:       funcs,
		Vars:        vars,
	})
	if err != nil {
		return nil, err
	}

	c := &Controller{
		env:  env,
		conf: conf,
	}

	return c, nil
}

func (c *Controller) init(ctx context.Context) (err error) {
	if c.stagesManager != nil {
		return fmt.Errorf("controller already started")
	}

	switch {
	case c.conf.ManageSingleNode != "":
		c.managePodsWithFieldSelector = fields.OneTermEqualSelector("spec.nodeName", c.conf.ManageSingleNode).String()
		c.manageNodesWithFieldSelector = fields.OneTermEqualSelector("metadata.name", c.conf.ManageSingleNode).String()
		c.manageNodeLeasesWithFieldSelector = fields.OneTermEqualSelector("metadata.name", c.conf.ManageSingleNode).String()
	case c.conf.ManageAllNodes:
		c.managePodsWithFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
	case c.conf.ManageNodesWithLabelSelector != "" || c.conf.ManageNodesWithAnnotationSelector != "":
		c.manageNodesWithLabelSelector = c.conf.ManageNodesWithLabelSelector
		c.manageNodesWithAnnotationSelector = c.conf.ManageNodesWithAnnotationSelector
		c.managePodsWithFieldSelector = fields.OneTermNotEqualSelector("spec.nodeName", "").String()
	}

	c.broadcaster = record.NewBroadcaster()
	c.recorder = c.broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "kwok_controller"})
	c.broadcaster.StartRecordingToSink(&clientcorev1.EventSinkImpl{Interface: c.conf.TypedClient.CoreV1().Events("")})

	c.nodesChan = make(chan informer.Event[*corev1.Node], 1)
	c.podsChan = make(chan informer.Event[*corev1.Pod], 1)

	nodesCli := c.conf.TypedClient.CoreV1().Nodes()
	c.nodesInformer = informer.NewInformer[*corev1.Node, *corev1.NodeList](nodesCli)
	c.nodeCacheGetter, err = c.nodesInformer.WatchWithCache(ctx, informer.Option{
		LabelSelector:      c.manageNodesWithLabelSelector,
		AnnotationSelector: c.manageNodesWithAnnotationSelector,
		FieldSelector:      c.manageNodesWithFieldSelector,
	}, c.nodesChan)
	if err != nil {
		return fmt.Errorf("failed to watch nodes: %w", err)
	}

	podsCli := c.conf.TypedClient.CoreV1().Pods(corev1.NamespaceAll)
	c.podsInformer = informer.NewInformer[*corev1.Pod, *corev1.PodList](podsCli)

	podWatchOption := informer.Option{
		FieldSelector: c.managePodsWithFieldSelector,
	}
	if c.conf.EnablePodCache {
		c.podCacheGetter, err = c.podsInformer.WatchWithLazyCache(ctx, podWatchOption, c.podsChan)
	} else {
		err = c.podsInformer.Watch(ctx, podWatchOption, c.podsChan)
	}
	if err != nil {
		return fmt.Errorf("failed to watch pods: %w", err)
	}

	if c.conf.NodeLeaseDurationSeconds != 0 {
		nodeLeasesCli := c.conf.TypedClient.CoordinationV1().Leases(corev1.NamespaceNodeLease)
		c.nodeLeasesInformer = informer.NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](nodeLeasesCli)
	}

	c.patchMeta = patch.NewPatchMetaFromOpenAPI3(c.conf.RESTClient)

	c.podOnNodeManageQueue = queue.NewQueue[string]()
	c.nodeManageQueue = queue.NewQueue[string]()
	return nil
}

func (c *Controller) initNodeLeaseController(ctx context.Context) error {
	if c.conf.NodeLeaseDurationSeconds == 0 {
		// Manage pods ignores leases
		c.onNodeManagedFunc = func(nodeName string) {
			c.podOnNodeManageQueue.Add(nodeName)
		}
		return nil
	}

	var err error
	c.nodeLeaseCacheGetter, err = c.nodeLeasesInformer.WatchWithCache(ctx, informer.Option{
		FieldSelector: c.manageNodeLeasesWithFieldSelector,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to watch node leases: %w", err)
	}

	leaseDuration := time.Duration(c.conf.NodeLeaseDurationSeconds) * time.Second
	// https://github.com/kubernetes/kubernetes/blob/02f4d643eae2e225591702e1bbf432efea453a26/pkg/kubelet/kubelet.go#L199-L200
	renewInterval := leaseDuration / 4
	// https://github.com/kubernetes/component-helpers/blob/d17b6f1e84500ee7062a26f5327dc73cb3e9374a/apimachinery/lease/controller.go#L100
	renewIntervalJitter := 0.04
	c.nodeLeases, err = NewNodeLeaseController(NodeLeaseControllerConfig{
		Clock:                c.conf.Clock,
		TypedClient:          c.conf.TypedClient,
		LeaseDurationSeconds: c.conf.NodeLeaseDurationSeconds,
		LeaseParallelism:     c.conf.NodeLeaseParallelism,
		GetLease: func(nodeName string) (*coordinationv1.Lease, bool) {
			return c.nodeLeaseCacheGetter.GetWithNamespace(nodeName, corev1.NamespaceNodeLease)
		},
		RenewInterval:       renewInterval,
		RenewIntervalJitter: renewIntervalJitter,
		MutateLeaseFunc: setNodeOwnerFunc(func(nodeName string) []metav1.OwnerReference {
			node, ok := c.nodeCacheGetter.Get(nodeName)
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
		HolderIdentity: c.conf.ID,
		OnNodeManagedFunc: func(nodeName string) {
			c.nodeManageQueue.Add(nodeName)
			c.podOnNodeManageQueue.Add(nodeName)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create node leases controller: %w", err)
	}

	// Not holding the lease means the node is not managed
	c.readOnlyFunc = func(nodeName string) bool {
		return !c.nodeLeases.Held(nodeName)
	}

	c.onNodeManagedFunc = func(nodeName string) {
		// Try to hold the lease
		c.nodeLeases.TryHold(nodeName)
	}
	c.onNodeUnmanagedFunc = func(nodeName string) {
		c.nodeLeases.ReleaseHold(nodeName)
	}

	go c.nodeLeaseSyncWorker(ctx)

	err = c.nodeLeases.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start node leases controller: %w", err)
	}
	return nil
}

func (c *Controller) nodeLeaseSyncWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		nodeName, ok := c.nodeManageQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		node, ok := c.nodeCacheGetter.Get(nodeName)
		if !ok {
			logger.Warn("node not found in cache", "node", nodeName)
			err := c.nodesInformer.Sync(ctx, informer.Option{
				FieldSelector: fields.OneTermEqualSelector("metadata.name", nodeName).String(),
			}, c.nodesChan)
			if err != nil {
				logger.Error("failed to update node", err, "node", nodeName)
			}
			continue
		}

		// Avoid slow cache synchronization, which may be judged as unmanaged.
		c.nodes.ManageNode(node)
	}
}

var podRef = internalversion.StageResourceRef{APIGroup: "v1", Kind: "Pod"}
var nodeRef = internalversion.StageResourceRef{APIGroup: "v1", Kind: "Node"}

func (c *Controller) startStageController(ctx context.Context, ref internalversion.StageResourceRef, lifecycle resources.Getter[lifecycle.Lifecycle]) error {
	switch ref {
	case podRef:
		err := c.initPodController(ctx, lifecycle)
		if err != nil {
			return fmt.Errorf("failed to init pod controller: %w", err)
		}

		for i := uint(0); i < c.conf.PodsOnNodeSyncParallelism; i++ {
			go c.podsOnNodeSyncWorker(ctx)
		}

	case nodeRef:
		err := c.initNodeLeaseController(ctx)
		if err != nil {
			return fmt.Errorf("failed to init node lease controller: %w", err)
		}

		err = c.initNodeController(ctx, lifecycle)
		if err != nil {
			return fmt.Errorf("failed to init node controller: %w", err)
		}
	default:
		err := c.initStageController(ctx, ref, lifecycle)
		if err != nil {
			return fmt.Errorf("failed to init stage controller: %w", err)
		}
	}
	return nil
}

func (c *Controller) initStagesManager(ctx context.Context) error {
	logger := log.FromContext(ctx)

	c.stageGetter = resources.NewDynamicGetter[
		[]*internalversion.Stage,
		*v1alpha1.Stage,
		*v1alpha1.StageList,
	](
		c.conf.TypedKwokClient.KwokV1alpha1().Stages(),
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

	err := c.stageGetter.Start(ctx)
	if err != nil {
		return err
	}

	stagesManager := NewStagesManager(StagesManagerConfig{
		StartFunc:   c.startStageController,
		StageGetter: c.stageGetter,
	})

	err = stagesManager.Start(ctx)
	if err != nil {
		return err
	}

	c.stagesManager = stagesManager
	return nil
}

func (c *Controller) onNodeManaged(nodeName string) {
	if c.onNodeManagedFunc == nil {
		return
	}
	c.onNodeManagedFunc(nodeName)
}

func (c *Controller) onNodeUnmanaged(nodeName string) {
	if c.onNodeUnmanagedFunc == nil {
		return
	}
	c.onNodeUnmanagedFunc(nodeName)
}

func (c *Controller) initNodeController(ctx context.Context, lifecycle resources.Getter[lifecycle.Lifecycle]) (err error) {
	c.nodes, err = NewNodeController(NodeControllerConfig{
		Clock:                                 c.conf.Clock,
		TypedClient:                           c.conf.TypedClient,
		NodeIP:                                c.conf.NodeIP,
		NodeName:                              c.conf.NodeName,
		NodePort:                              c.conf.NodePort,
		DisregardStatusWithAnnotationSelector: c.conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      c.conf.DisregardStatusWithLabelSelector,
		OnNodeManagedFunc:                     c.onNodeManaged,
		OnNodeUnmanagedFunc:                   c.onNodeUnmanaged,
		Lifecycle:                             lifecycle,
		PlayStageParallelism:                  c.conf.NodePlayStageParallelism,
		FuncMap:                               c.conf.FuncMap,
		Recorder:                              c.recorder,
		ReadOnlyFunc:                          c.readOnlyFunc,
		EnableMetrics:                         c.conf.EnableMetrics,
	})
	if err != nil {
		return fmt.Errorf("failed to create nodes controller: %w", err)
	}
	err = c.nodes.Start(ctx, c.nodesChan)
	if err != nil {
		return fmt.Errorf("failed to start nodes controller: %w", err)
	}

	return nil
}
func (c *Controller) initPodController(ctx context.Context, lifecycle resources.Getter[lifecycle.Lifecycle]) (err error) {
	c.pods, err = NewPodController(PodControllerConfig{
		Clock:                                 c.conf.Clock,
		EnableCNI:                             c.conf.EnableCNI,
		TypedClient:                           c.conf.TypedClient,
		NodeCacheGetter:                       c.nodeCacheGetter,
		NodeIP:                                c.conf.NodeIP,
		CIDR:                                  c.conf.CIDR,
		DisregardStatusWithAnnotationSelector: c.conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      c.conf.DisregardStatusWithLabelSelector,
		Lifecycle:                             lifecycle,
		PlayStageParallelism:                  c.conf.PodPlayStageParallelism,
		NodeGetFunc: func(nodeName string) (*NodeInfo, bool) {
			if c.nodes == nil {
				return nil, false
			}

			return c.nodes.Get(nodeName)
		},
		FuncMap:       c.conf.FuncMap,
		Recorder:      c.recorder,
		ReadOnlyFunc:  c.readOnlyFunc,
		EnableMetrics: c.conf.EnableMetrics,
	})
	if err != nil {
		return fmt.Errorf("failed to create pods controller: %w", err)
	}

	err = c.pods.Start(ctx, c.podsChan)
	if err != nil {
		return fmt.Errorf("failed to start pods controller: %w", err)
	}

	return nil
}

func (c *Controller) initStageController(ctx context.Context, ref internalversion.StageResourceRef, lifecycle resources.Getter[lifecycle.Lifecycle]) error {
	logger := log.FromContext(ctx)

	gv, err := schema.ParseGroupVersion(ref.APIGroup)
	if err != nil {
		return fmt.Errorf("failed to parse group version: %w", err)
	}

	gvr, err := c.conf.RESTMapper.ResourceFor(gv.WithResource(ref.Kind))
	if err != nil {
		return fmt.Errorf("failed to get gvk for gvr: %w", err)
	}

	schema, err := c.patchMeta.Lookup(gvr)
	if err != nil {
		return err
	}

	logger.Info("watching stages", "gvr", gvr)
	stageInformer := informer.NewInformer[*unstructured.Unstructured, *unstructured.UnstructuredList](c.conf.DynamicClient.Resource(gvr))
	stageChan := make(chan informer.Event[*unstructured.Unstructured], 1)
	err = stageInformer.Watch(ctx, informer.Option{}, stageChan)
	if err != nil {
		return fmt.Errorf("failed to watch stages: %w", err)
	}

	stage, err := NewStageController(StageControllerConfig{
		Clock:                                 c.conf.Clock,
		DynamicClient:                         c.conf.DynamicClient,
		ImpersonatingDynamicClient:            c.conf.ImpersonatingDynamicClient,
		Schema:                                schema,
		GVR:                                   gvr,
		DisregardStatusWithAnnotationSelector: c.conf.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      c.conf.DisregardStatusWithLabelSelector,
		Lifecycle:                             lifecycle,
		PlayStageParallelism:                  1,
		FuncMap:                               c.conf.FuncMap,
		Recorder:                              c.recorder,
	})
	if err != nil {
		return fmt.Errorf("failed to create stage controller: %w", err)
	}

	err = stage.Start(ctx, stageChan)
	if err != nil {
		return fmt.Errorf("failed to start stage controller: %w", err)
	}

	return nil
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	err := c.init(ctx)
	if err != nil {
		return fmt.Errorf("failed to init controller: %w", err)
	}

	if len(c.conf.LocalStages) != 0 {
		for ref, stage := range c.conf.LocalStages {
			lifecycle, err := lifecycle.NewLifecycle(stage, c.env)
			if err != nil {
				return err
			}
			err = c.startStageController(ctx, ref, resources.NewStaticGetter(lifecycle))
			if err != nil {
				return err
			}
		}
	} else {
		err = c.initStagesManager(ctx)
		if err != nil {
			return fmt.Errorf("failed to init stages manager: %w", err)
		}
	}
	return nil
}

func (c *Controller) podsOnNodeSyncWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		nodeName, ok := c.podOnNodeManageQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}

		opt := informer.Option{
			FieldSelector:     fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
			EnableListPager:   c.conf.EnablePodsOnNodeSyncListPager,
			EnableStreamWatch: c.conf.EnablePodsOnNodeSyncStreamWatch,
		}

		err := c.podsInformer.Sync(ctx, opt, c.podsChan)
		if err != nil {
			logger.Error("failed to update pods on node", err, "node", nodeName)
		}
	}
}

// ListNodes returns all nodes
func (c *Controller) ListNodes() []string {
	if c.nodes == nil {
		return nil
	}
	return c.nodes.List()
}

// ListPods returns all pods on the given node
func (c *Controller) ListPods(nodeName string) ([]log.ObjectRef, bool) {
	if c.pods == nil {
		return nil, false
	}
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
	if c.nodes == nil {
		return 0
	}
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
