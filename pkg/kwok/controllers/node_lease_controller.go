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
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/queue"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// NodeLeaseController is responsible for creating and renewing a lease object
type NodeLeaseController struct {
	typedClient          clientset.Interface
	nodeCacheGetter      informer.Getter[*corev1.Node]
	leaseDurationSeconds uint
	leaseParallelism     uint
	renewInterval        time.Duration
	renewIntervalJitter  float64
	clock                clock.Clock

	// latestLease is the latest lease which the NodeLeaseController updated or created
	latestLease maps.SyncMap[string, *coordinationv1.Lease]

	// mutateLeaseFunc allows customizing a lease object
	mutateLeaseFunc func(*coordinationv1.Lease) error

	delayQueue queue.DelayingQueue[string]

	holderIdentity    string
	onNodeManagedFunc func(nodeName string)
}

// NodeLeaseControllerConfig is the configuration for NodeLeaseController
type NodeLeaseControllerConfig struct {
	Clock                clock.Clock
	HolderIdentity       string
	TypedClient          clientset.Interface
	NodeCacheGetter      informer.Getter[*corev1.Node]
	LeaseDurationSeconds uint
	LeaseParallelism     uint
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
		nodeCacheGetter:      conf.NodeCacheGetter,
		leaseDurationSeconds: conf.LeaseDurationSeconds,
		leaseParallelism:     conf.LeaseParallelism,
		renewInterval:        conf.RenewInterval,
		renewIntervalJitter:  conf.RenewIntervalJitter,
		mutateLeaseFunc:      conf.MutateLeaseFunc,
		delayQueue:           queue.NewDelayingQueue[string](conf.Clock),
		holderIdentity:       conf.HolderIdentity,
		onNodeManagedFunc:    conf.OnNodeManagedFunc,
	}

	return c, nil
}

// Start starts the NodeLeaseController
func (c *NodeLeaseController) Start(ctx context.Context, events <-chan informer.Event[*coordinationv1.Lease]) error {
	for i := uint(0); i < c.leaseParallelism; i++ {
		go c.syncWorker(ctx)
	}
	go c.watchResources(ctx, events)
	return nil
}

// watchResources watch resources and send to preprocessChan
func (c *NodeLeaseController) watchResources(ctx context.Context, events <-chan informer.Event[*coordinationv1.Lease]) {
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
				lease := event.Object.DeepCopy()
				c.latestLease.Store(lease.Name, lease)
			case informer.Deleted:
				lease := event.Object
				c.remove(lease.Name)
			}
		case <-ctx.Done():
			break loop
		}
	}
	logger.Info("Stop watch node leases")
}

func (c *NodeLeaseController) syncWorker(ctx context.Context) {
	for ctx.Err() == nil {
		nodeName := c.delayQueue.GetOrWait()
		if c.nodeCacheGetter != nil {
			_, ok := c.nodeCacheGetter.Get(nodeName)
			if !ok {
				continue
			}
		}

		now := c.clock.Now()
		c.sync(ctx, nodeName)
		nextTime := c.nextTryTime(nodeName, now)
		_ = c.delayQueue.AddAfter(nodeName, nextTime.Sub(now))
	}
}

func (c *NodeLeaseController) nextTryTime(name string, now time.Time) time.Time {
	next := now.Add(wait.Jitter(c.renewInterval, c.renewIntervalJitter))
	lease, ok := c.latestLease.Load(name)
	if !ok || lease == nil ||
		lease.Spec.HolderIdentity == nil ||
		lease.Spec.LeaseDurationSeconds == nil ||
		lease.Spec.RenewTime == nil {
		return next
	}
	return nextTryTime(lease, c.holderIdentity, next)
}

// TryHold tries to hold a lease for the NodeLeaseController
func (c *NodeLeaseController) TryHold(name string) {
	c.delayQueue.Add(name)
}

// remove removes a lease from the NodeLeaseController
func (c *NodeLeaseController) remove(name string) {
	_ = c.delayQueue.Cancel(name)
	c.latestLease.Delete(name)
}

// Held returns true if the NodeLeaseController holds the lease
func (c *NodeLeaseController) Held(name string) bool {
	lease, ok := c.latestLease.Load(name)
	if !ok || lease == nil || lease.Spec.HolderIdentity == nil {
		return false
	}

	return *lease.Spec.HolderIdentity == c.holderIdentity
}

// sync syncs a lease for a node
func (c *NodeLeaseController) sync(ctx context.Context, nodeName string) {
	logger := log.FromContext(ctx)
	logger = logger.With("node", nodeName)

	latestLease, ok := c.latestLease.Load(nodeName)
	if ok && latestLease != nil {
		if !tryAcquireOrRenew(latestLease, c.holderIdentity, c.clock.Now()) {
			logger.Debug("Lease already acquired by another holder")
			return
		}
		logger.Info("Syncing lease")
		lease, transitions, err := c.renewLease(ctx, latestLease)
		if err != nil {
			logger.Error("failed to update lease using latest lease", err)
			return
		}
		c.latestLease.Store(nodeName, lease)
		if transitions {
			logger.Debug("Lease transitioned",
				"transitions", transitions,
			)
			if c.onNodeManagedFunc != nil {
				if c.Held(nodeName) {
					c.onNodeManagedFunc(nodeName)
				} else {
					logger.Warn("Lease not held")
				}
			}
		}
	} else {
		logger.Info("Creating lease")
		latestLease, err := c.ensureLease(ctx, nodeName)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Error("failed to create lease, lease already exists", err)

				_, err = c.syncLease(ctx, nodeName)
				if err != nil {
					logger.Error("failed to sync lease", err)
					return
				}
				if c.onNodeManagedFunc != nil {
					if c.Held(nodeName) {
						c.onNodeManagedFunc(nodeName)
					} else {
						logger.Warn("Lease not held")
					}
				}
				return
			}

			if !apierrors.IsNotFound(err) || !c.latestLease.IsEmpty() {
				logger.Error("failed to create lease", err)
				return
			}

			// kube-apiserver will not have finished initializing the resources when the cluster has just been created.
			logger.Error("lease namespace not found, retrying in 1 second", err)
			c.clock.Sleep(1 * time.Second)
			latestLease, err = c.ensureLease(ctx, nodeName)
			if err != nil {
				logger.Error("failed to create lease secondly", err)
				return
			}
		}

		c.latestLease.Store(nodeName, latestLease)
		if c.onNodeManagedFunc != nil {
			if c.Held(nodeName) {
				c.onNodeManagedFunc(nodeName)
			} else {
				logger.Warn("Lease not held")
			}
		}
	}
}

func (c *NodeLeaseController) syncLease(ctx context.Context, leaseName string) (*coordinationv1.Lease, error) {
	lease, err := c.typedClient.CoordinationV1().Leases(corev1.NamespaceNodeLease).Get(ctx, leaseName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	c.latestLease.Store(leaseName, lease)
	return lease, nil
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
