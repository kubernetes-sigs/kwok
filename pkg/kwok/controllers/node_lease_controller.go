/*
Copyright 2023 The Kubernetes Authors.

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
	"time"

	"github.com/wzshiming/cron"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	informers "k8s.io/client-go/informers/coordination/v1"
	clientset "k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/coordination/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// NodeLeaseController is responsible for creating and renewing a lease object
type NodeLeaseController struct {
	typedClient clientset.Interface

	leaseLister   listers.LeaseLister
	leaseInformer cache.SharedIndexInformer
	leasesSynced  cache.InformerSynced

	leaseNamespace       string
	leaseDurationSeconds uint
	leaseParallelism     uint
	renewInterval        time.Duration
	renewIntervalJitter  float64
	clock                clock.Clock

	// mutateLeaseFunc allows customizing a lease object
	mutateLeaseFunc func(*coordinationv1.Lease) error

	cronjob   *cron.Cron
	cancelJob maps.SyncMap[string, cron.DoFunc]

	leaseChan chan string

	holderIdentity    string
	onNodeManagedFunc func(nodeName string)
}

// NodeLeaseControllerConfig is the configuration for NodeLeaseController
type NodeLeaseControllerConfig struct {
	Clock                clock.Clock
	HolderIdentity       string
	TypedClient          clientset.Interface
	LeaseDurationSeconds uint
	LeaseParallelism     uint
	RenewInterval        time.Duration
	RenewIntervalJitter  float64
	LeaseNamespace       string
	MutateLeaseFunc      func(*coordinationv1.Lease) error
	OnNodeManagedFunc    func(nodeName string)
}

// NewNodeLeaseController constructs and returns a NodeLeaseController
func NewNodeLeaseController(conf NodeLeaseControllerConfig) (*NodeLeaseController, error) {
	if conf.LeaseParallelism <= 0 {
		return nil, fmt.Errorf("node leases parallelism must be greater than 0")
	}

	if conf.Clock == nil {
		conf.Clock = clock.RealClock{}
	}

	leaseInformer := informers.NewFilteredLeaseInformer(
		conf.TypedClient,
		conf.LeaseNamespace,
		0,
		cache.Indexers{
			cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
		},
		func(listOptions *metav1.ListOptions) {
			listOptions.AllowWatchBookmarks = true
		},
	)

	c := &NodeLeaseController{
		clock:                conf.Clock,
		typedClient:          conf.TypedClient,
		leaseInformer:        leaseInformer,
		leaseLister:          listers.NewLeaseLister(leaseInformer.GetIndexer()),
		leasesSynced:         leaseInformer.HasSynced,
		leaseNamespace:       conf.LeaseNamespace,
		leaseDurationSeconds: conf.LeaseDurationSeconds,
		leaseParallelism:     conf.LeaseParallelism,
		renewInterval:        conf.RenewInterval,
		renewIntervalJitter:  conf.RenewIntervalJitter,
		mutateLeaseFunc:      conf.MutateLeaseFunc,
		cronjob:              cron.NewCron(),
		leaseChan:            make(chan string),
		holderIdentity:       conf.HolderIdentity,
		onNodeManagedFunc:    conf.OnNodeManagedFunc,
	}

	if _, err := leaseInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteLeaseEventHandler,
	}); err != nil {
		return nil, fmt.Errorf("failed to add delete event handler for node lease: %w", err)
	}

	return c, nil
}

// Start starts the NodeLeaseController
func (c *NodeLeaseController) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting node lease controller")

	go c.leaseInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.leasesSynced) {
		return fmt.Errorf("timed out waiting for node lease caches to sync")
	}

	for i := uint(0); i < c.leaseParallelism; i++ {
		go c.syncWorker(ctx)
	}
	return nil
}

func (c *NodeLeaseController) syncWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Stop sync worker")
			return
		case nodeName := <-c.leaseChan:
			c.sync(ctx, nodeName)
		}
	}
}

func (c *NodeLeaseController) nextTryTime(leaseName string, now time.Time) time.Time {
	next := now.Add(wait.Jitter(c.renewInterval, c.renewIntervalJitter))

	lease, err := c.leaseLister.Leases(c.leaseNamespace).Get(leaseName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			panic(fmt.Errorf("unexpected error during get lease %s: %w", leaseName, err))
		}
	}

	if lease == nil ||
		lease.Spec.HolderIdentity == nil ||
		lease.Spec.LeaseDurationSeconds == nil ||
		lease.Spec.RenewTime == nil {
		return next
	}
	return nextTryTime(lease, c.holderIdentity, next)
}

// TryHold tries to hold a lease for the NodeLeaseController
func (c *NodeLeaseController) TryHold(name string) {
	// trigger a sync immediately
	c.leaseChan <- name

	// add a cron job to sync the lease periodically
	cancel, ok := c.cronjob.AddWithCancel(
		func(now time.Time) (time.Time, bool) {
			return c.nextTryTime(name, now), true
		},
		func() {
			c.leaseChan <- name
		},
	)
	if ok {
		old, ok := c.cancelJob.LoadOrStore(name, cancel)
		if ok {
			old()
		}
	}
}

func (c *NodeLeaseController) deleteLeaseEventHandler(obj interface{}) {
	lease, ok := obj.(*coordinationv1.Lease)

	// When a delete is dropped, the relist will notice a lease in the store
	// not in the list, leading to the insertion of a tombstone object which
	// contains the deleted key/value. Note that this value might be stale.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
			return
		}
		lease, ok = tombstone.Obj.(*coordinationv1.Lease)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a lease %#v", obj))
			return
		}
	}

	cancel, ok := c.cancelJob.LoadAndDelete(lease.Name)
	if ok {
		cancel()
	}
}

// Held returns true if the NodeLeaseController holds the lease
func (c *NodeLeaseController) Held(name string) bool {
	lease, err := c.leaseLister.Leases(c.leaseNamespace).Get(name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			panic(fmt.Errorf("unexpected error during get lease %s: %w", name, err))
		}
	}

	if lease == nil || lease.Spec.HolderIdentity == nil {
		return false
	}
	return *lease.Spec.HolderIdentity == c.holderIdentity
}

// sync syncs a lease for a node
func (c *NodeLeaseController) sync(ctx context.Context, name string) {
	logger := log.FromContext(ctx)

	logger = logger.With("node", name)

	lease, err := c.leaseLister.Leases(c.leaseNamespace).Get(name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			panic(fmt.Errorf("unexpected error during get lease %s: %w", name, err))
		}
	}

	if lease == nil {
		logger.Info("Creating lease")

		_, err = c.ensureLease(ctx, name)
		if err != nil {
			if !apierrors.IsAlreadyExists(err) {
				panic(fmt.Errorf("unexpected error during create lease %s: %w", name, err))
			}
			logger.Error("lease %s is already exist, the cache might out of date. Let's try it later", err, name)
			return
		}

		if c.onNodeManagedFunc != nil {
			c.onNodeManagedFunc(name)
		}
		return
	}

	if !tryAcquireOrRenew(lease, c.holderIdentity, c.clock.Now()) {
		logger.Debug("Lease already acquired by another holder")
		return
	}

	logger.Info("Syncing lease")
	lease, transitions, err := c.renewLease(ctx, lease)
	if err != nil {
		logger.Error("failed to update lease using latest lease", err)
		return
	}

	if transitions {
		logger.Info("Lease transitioned", "transitions", transitions)
		if c.onNodeManagedFunc != nil {
			c.onNodeManagedFunc(name)
		}
	}
}

// ensureLease creates a lease if it does not exist
func (c *NodeLeaseController) ensureLease(ctx context.Context, leaseName string) (*coordinationv1.Lease, error) {
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: c.leaseNamespace,
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &c.holderIdentity,
			LeaseDurationSeconds: format.Ptr(int32(c.leaseDurationSeconds)),
			RenewTime:            format.Ptr(metav1.NewMicroTime(c.clock.Now())),
		},
	}
	if c.mutateLeaseFunc != nil {
		err := c.mutateLeaseFunc(lease)
		if err != nil {
			return nil, err
		}
	}

	lease, err := c.typedClient.CoordinationV1().Leases(c.leaseNamespace).Create(ctx, lease, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return lease, nil
}

// renewLease attempts to update the lease for maxUpdateRetries, call this once you're sure the lease has been created
func (c *NodeLeaseController) renewLease(ctx context.Context, base *coordinationv1.Lease) (*coordinationv1.Lease, bool, error) {
	lease := base.DeepCopy()

	transitions := format.ElemOrDefault(lease.Spec.HolderIdentity) != c.holderIdentity
	if transitions {
		lease.Spec.HolderIdentity = &c.holderIdentity
		lease.Spec.LeaseDurationSeconds = format.Ptr(int32(c.leaseDurationSeconds))
		lease.Spec.LeaseTransitions = format.Ptr(format.ElemOrDefault(lease.Spec.LeaseTransitions) + 1)
	}
	lease.Spec.RenewTime = format.Ptr(metav1.NewMicroTime(c.clock.Now()))

	if c.mutateLeaseFunc != nil {
		err := c.mutateLeaseFunc(lease)
		if err != nil {
			return nil, false, err
		}
	}

	lease, err := c.typedClient.CoordinationV1().Leases(c.leaseNamespace).Update(ctx, lease, metav1.UpdateOptions{})
	if err != nil {
		return nil, false, err
	}
	return lease, transitions, nil
}

// setNodeOwnerFunc helps construct a mutateLeaseFunc which sets a node OwnerReference to the given lease object
// https://github.com/kubernetes/kubernetes/blob/1f22a173d9538e01c92529d02e4c95f77f5ea823/pkg/kubelet/util/nodelease.go#L32
func setNodeOwnerFunc(nodeOwnerFunc func(nodeName string) []metav1.OwnerReference) func(lease *coordinationv1.Lease) error {
	return func(lease *coordinationv1.Lease) error {
		// Setting owner reference needs node's UID. Note that it is different from
		// kubelet.nodeRef.UID. When lease is initially created, it is possible that
		// the connection between master and node is not ready yet. So try to set
		// owner reference every time when renewing the lease, until successful.
		if len(lease.OwnerReferences) == 0 {
			lease.OwnerReferences = nodeOwnerFunc(lease.Name)
		}
		return nil
	}
}

// tryAcquireOrRenew returns true if the lease is held by the given holderIdentity,
func tryAcquireOrRenew(lease *coordinationv1.Lease, holderIdentity string, now time.Time) bool {
	if lease.Spec.HolderIdentity == nil ||
		*lease.Spec.HolderIdentity == holderIdentity {
		return true
	}

	if lease.Spec.RenewTime == nil ||
		lease.Spec.LeaseDurationSeconds == nil {
		return true
	}

	expireTime := lease.Spec.RenewTime.Add(time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second)
	return expireTime.Before(now)
}

// nextTryTime returns the next time to try to acquire or renew the lease
func nextTryTime(lease *coordinationv1.Lease, holderIdentity string, next time.Time) time.Time {
	expireTime := lease.Spec.RenewTime.Add(time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second)
	if *lease.Spec.HolderIdentity == holderIdentity {
		if next.Before(expireTime) {
			return next
		}
	}

	return expireTime
}
