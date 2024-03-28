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
	holdLeaseSet maps.SyncMap[string, *corev1.Node]

	holderIdentity    string
	onNodeManagedFunc func(nodeName string, node *corev1.Node)
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
	OnNodeManagedFunc    func(nodeName string, node *corev1.Node)
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
		_, ok = c.holdLeaseSet.Load(nodeName)
		if !ok {
			continue
		}

		dur := c.interval()

		lease, err := c.sync(ctx, nodeName)
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

		now := c.clock.Now()
		expireDuration := expireTime.Sub(now)
		hold := tryAcquireOrRenew(lease, c.holderIdentity, now)
		nextTry := nextTryDuration(dur, expireDuration, hold)
		c.delayQueue.AddWeightAfter(nodeName, 2, nextTry)
	}
}

func (c *NodeLeaseController) interval() time.Duration {
	return wait.Jitter(c.renewInterval, c.renewIntervalJitter)
}

// TryHold tries to hold a lease for the NodeLeaseController
func (c *NodeLeaseController) TryHold(nodeName string, node *corev1.Node) {
	_, loaded := c.holdLeaseSet.Swap(nodeName, node)
	if !loaded {
		c.delayQueue.Add(nodeName)
	}
}

// ReleaseHold releases a lease for the NodeLeaseController
func (c *NodeLeaseController) ReleaseHold(nodeName string) {
	_ = c.delayQueue.Cancel(nodeName)
	c.holdLeaseSet.Delete(nodeName)
}

// Held returns true if the NodeLeaseController holds the lease
func (c *NodeLeaseController) Held(name string) bool {
	lease, ok := c.getLease(name)
	if !ok || lease == nil || lease.Spec.HolderIdentity == nil {
		return false
	}

	return *lease.Spec.HolderIdentity == c.holderIdentity
}

// sync syncs a lease for a node
func (c *NodeLeaseController) sync(ctx context.Context, nodeName string) (lease *coordinationv1.Lease, err error) {
	logger := log.FromContext(ctx)
	logger = logger.With("node", nodeName)

	lease, _ = c.getLease(nodeName)
	if lease != nil {
		if !tryAcquireOrRenew(lease, c.holderIdentity, c.clock.Now()) {
			logger.Debug("Lease already acquired by another holder")
			return nil, nil
		}
		logger.Info("Syncing lease")
		lease, err := c.renewLease(ctx, lease)
		if err != nil {
			return nil, fmt.Errorf("failed to update lease using lease: %w", err)
		}

		c.onNodeManaged(nodeName)
		return lease, nil
	}

	logger.Info("Creating lease")
	lease, err = c.ensureLease(ctx, nodeName)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create lease, lease already exists: %w", err)
		}

		// kube-apiserver will not have finished initializing the resources when the cluster has just been created.
		for apierrors.IsNotFound(err) {
			logger.Error("lease namespace not found, retrying in 1 second", err)
			c.clock.Sleep(1 * time.Second)
			lease, err = c.ensureLease(ctx, nodeName)
		}
		if err != nil {
			return lease, fmt.Errorf("failed to create lease after retrying: %w", err)
		}
	}

	c.onNodeManaged(nodeName)
	return lease, nil
}

func (c *NodeLeaseController) onNodeManaged(nodeName string) {
	if c.onNodeManagedFunc == nil {
		return
	}

	node, _ := c.holdLeaseSet.Swap(nodeName, nil)
	if node != nil {
		c.onNodeManagedFunc(nodeName, node)
	}
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
func (c *NodeLeaseController) renewLease(ctx context.Context, base *coordinationv1.Lease) (*coordinationv1.Lease, error) {
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
			return nil, err
		}
	}

	lease, err := c.typedClient.CoordinationV1().Leases(lease.Namespace).Update(ctx, lease, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return lease, nil
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
