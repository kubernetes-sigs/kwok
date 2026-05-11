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

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// cacheSyncRetryDelay is the interval between polls of the informer cache
// while waiting for a newly acquired/transitioned lease to be reflected.
const cacheSyncRetryDelay = 50 * time.Millisecond

// NodeLeaseController is responsible for creating and renewing a lease object
type NodeLeaseController struct {
	typedClient          clientset.Interface
	leaseDurationSeconds uint
	leaseParallelism     uint
	renewInterval        time.Duration
	renewIntervalJitter  float64
	clock                clock.Clock

	getLease func(nodeName string) (*coordinationv1.Lease, bool)

	// mutateLeaseFunc allows customizing a lease object
	mutateLeaseFunc func(*coordinationv1.Lease) error

	delayQueue   queue.WeightDelayingQueue[string]
	holdLeaseSet maps.SyncMap[string, bool]
	// pendingManageSet tracks nodes whose lease has been successfully acquired
	// or transitioned but whose ownership has not yet been confirmed by the
	// informer cache. The delay queue is reused to re-check the cache with a
	// short delay, avoiding a blocking spin-loop in syncWorker.
	pendingManageSet maps.SyncMap[string, struct{}]

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
	GetLease             func(nodeName string) (*coordinationv1.Lease, bool)
	RenewInterval        time.Duration
	RenewIntervalJitter  float64
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

	c := &NodeLeaseController{
		clock:                conf.Clock,
		typedClient:          conf.TypedClient,
		leaseDurationSeconds: conf.LeaseDurationSeconds,
		leaseParallelism:     conf.LeaseParallelism,
		getLease:             conf.GetLease,
		renewInterval:        conf.RenewInterval,
		renewIntervalJitter:  conf.RenewIntervalJitter,
		mutateLeaseFunc:      conf.MutateLeaseFunc,
		delayQueue:           queue.NewWeightDelayingQueue[string](conf.Clock),
		holderIdentity:       conf.HolderIdentity,
		onNodeManagedFunc:    conf.OnNodeManagedFunc,
	}

	return c, nil
}

// Start starts the NodeLeaseController
func (c *NodeLeaseController) Start(ctx context.Context) error {
	for i := uint(0); i < c.leaseParallelism; i++ {
		go c.syncWorker(ctx)
	}
	return nil
}

func (c *NodeLeaseController) syncWorker(ctx context.Context) {
	logger := log.FromContext(ctx)
	for ctx.Err() == nil {
		nodeName, ok := c.delayQueue.GetOrWaitWithDone(ctx.Done())
		if !ok {
			return
		}
		first, ok := c.holdLeaseSet.Load(nodeName)
		if !ok {
			continue
		}

		// If we previously acquired the lease but the informer cache hadn't
		// yet reflected our ownership, check again now before doing a full
		// sync (which would prematurely renew the lease).
		if _, pending := c.pendingManageSet.Load(nodeName); pending {
			if c.Held(nodeName) {
				// Cache now confirms our ownership; notify consumers and
				// schedule the next regular renewal from the cached lease.
				c.pendingManageSet.Delete(nodeName)
				c.onNodeManaged(nodeName)
				if cachedLease, ok := c.getLease(nodeName); ok {
					if expTime, ok := expireTime(cachedLease); ok {
						dur := c.interval()
						now := c.clock.Now()
						expireDuration := expTime.Sub(now)
						nextTry := nextTryDuration(dur, expireDuration, true)
						c.delayQueue.AddWeightAfter(nodeName, 2, nextTry)
						continue
					}
				}
				// Cache entry is gone or has no expire time; fall through
				// to a full sync so the lease gets re-created/renewed.
			} else {
				// Cache hasn't caught up yet; re-check after a short delay
				// without renewing the lease.
				c.delayQueue.AddWeightAfter(nodeName, 2, cacheSyncRetryDelay)
				continue
			}
		}

		dur := c.interval()

		lease, shouldManage, err := c.sync(ctx, nodeName, first)
		if err != nil {
			logger.Error("Failed to sync lease", err,
				"node", nodeName,
			)
			c.delayQueue.AddWeightAfter(nodeName, 1, dur)
			continue
		}

		expireTime, ok := expireTime(lease)
		if !ok {
			c.delayQueue.AddWeightAfter(nodeName, 1, dur)
			continue
		}

		if first {
			c.holdLeaseSet.Store(nodeName, false)
		}

		if shouldManage {
			// The lease was acquired or transitioned. Defer the onNodeManaged
			// notification until the informer cache confirms our ownership so
			// that Held() returns true when downstream code checks it.
			// Re-queue with a short delay to avoid a blocking spin-loop.
			c.pendingManageSet.Store(nodeName, struct{}{})
			c.delayQueue.AddWeightAfter(nodeName, 2, cacheSyncRetryDelay)
		} else {
			now := c.clock.Now()
			expireDuration := expireTime.Sub(now)
			hold := tryAcquireOrRenew(lease, c.holderIdentity, now)
			nextTry := nextTryDuration(dur, expireDuration, hold)
			c.delayQueue.AddWeightAfter(nodeName, 2, nextTry)
		}
	}
}

func (c *NodeLeaseController) interval() time.Duration {
	return wait.Jitter(c.renewInterval, c.renewIntervalJitter)
}

// TryHold tries to hold a lease for the NodeLeaseController
func (c *NodeLeaseController) TryHold(name string) {
	_, loaded := c.holdLeaseSet.LoadOrStore(name, true)
	if !loaded {
		c.delayQueue.Add(name)
	}
}

// ReleaseHold releases a lease for the NodeLeaseController
func (c *NodeLeaseController) ReleaseHold(name string) {
	_ = c.delayQueue.Cancel(name)
	c.holdLeaseSet.Delete(name)
	c.pendingManageSet.Delete(name)
}

// Held returns true if the NodeLeaseController holds the lease
func (c *NodeLeaseController) Held(name string) bool {
	lease, ok := c.getLease(name)
	if !ok || lease == nil || lease.Spec.HolderIdentity == nil {
		return false
	}

	return *lease.Spec.HolderIdentity == c.holderIdentity
}

// sync syncs a lease for a node.
// The returned boolean indicates whether onNodeManaged should be called for
// this node after the cache has been confirmed to reflect the new ownership.
func (c *NodeLeaseController) sync(ctx context.Context, nodeName string, first bool) (lease *coordinationv1.Lease, shouldManage bool, err error) {
	logger := log.FromContext(ctx)
	logger = logger.With("node", nodeName)

	lease, _ = c.getLease(nodeName)
	if lease != nil {
		if !tryAcquireOrRenew(lease, c.holderIdentity, c.clock.Now()) {
			logger.Debug("Lease already acquired by another holder")
			return nil, false, nil
		}
		logger.Info("Syncing lease")
		lease, transitions, err := c.renewLease(ctx, lease)
		if err != nil {
			return nil, false, fmt.Errorf("failed to update lease using lease: %w", err)
		}

		// it is first or it has been transitioned, and then manage the node.
		return lease, first || transitions, nil
	}

	logger.Info("Creating lease")
	lease, err = c.ensureLease(ctx, nodeName)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, false, fmt.Errorf("failed to create lease, lease already exists: %w", err)
		}

		// kube-apiserver will not have finished initializing the resources when the cluster has just been created.
		for apierrors.IsNotFound(err) {
			logger.Error("lease namespace not found, retrying in 1 second", err)
			c.clock.Sleep(1 * time.Second)
			lease, err = c.ensureLease(ctx, nodeName)
		}
		if err != nil {
			return lease, false, fmt.Errorf("failed to create lease after retrying: %w", err)
		}
	}

	return lease, true, nil
}

func (c *NodeLeaseController) onNodeManaged(nodeName string) {
	if c.onNodeManagedFunc == nil {
		return
	}

	c.onNodeManagedFunc(nodeName)
}

// ensureLease creates a lease if it does not exist
func (c *NodeLeaseController) ensureLease(ctx context.Context, leaseName string) (*coordinationv1.Lease, error) {
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: corev1.NamespaceNodeLease,
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

	lease, err := c.typedClient.CoordinationV1().Leases(corev1.NamespaceNodeLease).Create(ctx, lease, metav1.CreateOptions{})
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

	lease, err := c.typedClient.CoordinationV1().Leases(lease.Namespace).Update(ctx, lease, metav1.UpdateOptions{})
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

// expireTime returns the expire time of a lease
func expireTime(lease *coordinationv1.Lease) (time.Time, bool) {
	if lease == nil ||
		lease.Spec.HolderIdentity == nil ||
		lease.Spec.LeaseDurationSeconds == nil ||
		lease.Spec.RenewTime == nil {
		return time.Time{}, false
	}

	expireTime := lease.Spec.RenewTime.Add(time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second)
	return expireTime, true
}

// nextTryDuration returns the next to try to acquire or renew the lease
func nextTryDuration(renewInterval, expire time.Duration, hold bool) time.Duration {
	switch {
	case !hold:
		// If the lease is not held, we should retry at the renew interval.
		return renewInterval
	case renewInterval < expire:
		// If the lease is held and the renew interval is less than the expire time,
		// we should retry at the renew interval.
		return renewInterval
	case expire < time.Second:
		// If the lease is held and the expire time is less than 1 second,
		return time.Second
	default:
		// Otherwise, we should retry at the expire time.
		return expire
	}
}
