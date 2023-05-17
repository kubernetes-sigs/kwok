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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/pager"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// NodeLeaseController is responsible for creating and renewing a lease object
type NodeLeaseController struct {
	clientSet            clientset.Interface
	leaseNamespace       string
	leaseDurationSeconds uint
	leaseParallelism     uint
	renewInterval        time.Duration
	renewIntervalJitter  float64
	clock                clock.Clock

	// latestLease is the latest lease which the NodeLeaseController updated or created
	latestLease maps.SyncMap[string, *coordinationv1.Lease]

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
	ClientSet            clientset.Interface
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

	c := &NodeLeaseController{
		clock:                conf.Clock,
		clientSet:            conf.ClientSet,
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

	return c, nil
}

// Start starts the NodeLeaseController
func (c *NodeLeaseController) Start(ctx context.Context) error {
	for i := uint(0); i < c.leaseParallelism; i++ {
		go c.syncWorker(ctx)
	}

	opt := metav1.ListOptions{}
	err := c.watchResources(ctx, opt)
	if err != nil {
		return fmt.Errorf("failed watch node leases: %w", err)
	}

	logger := log.FromContext(ctx)
	go func() {
		err = c.listResources(ctx, opt)
		if err != nil {
			logger.Error("Failed list node leases", err)
		}
	}()
	return nil
}

// watchResources watch resources and send to preprocessChan
func (c *NodeLeaseController) watchResources(ctx context.Context, opt metav1.ListOptions) error {
	// Watch node leases in the cluster
	watcher, err := c.clientSet.CoordinationV1().Leases(c.leaseNamespace).Watch(ctx, opt)
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
						watcher, err := c.clientSet.CoordinationV1().Leases(c.leaseNamespace).Watch(ctx, opt)
						if err == nil {
							rc = watcher.ResultChan()
							continue loop
						}

						logger.Error("Failed to watch node leases", err)
						select {
						case <-ctx.Done():
							break loop
						case <-c.clock.After(time.Second * 5):
						}
					}
				}
				switch event.Type {
				case watch.Added, watch.Modified:
					lease := event.Object.(*coordinationv1.Lease)
					c.latestLease.Store(lease.Name, lease)
				case watch.Deleted:
					lease := event.Object.(*coordinationv1.Lease)
					c.remove(lease.Name)
				}
			case <-ctx.Done():
				watcher.Stop()
				break loop
			}
		}
		logger.Info("Stop watch node leases")
	}()
	return nil
}

// listResources lists all resources and sends to preprocessChan
func (c *NodeLeaseController) listResources(ctx context.Context, opt metav1.ListOptions) error {
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientSet.CoordinationV1().Leases(c.leaseNamespace).List(ctx, opts)
	})

	logger := log.FromContext(ctx)
	if logger.Enabled(ctx, log.LevelWarn) {
		defer log.Elapsed(ctx, c.clock, log.LevelWarn, time.Second, "Long time to list node leases")()
	}

	return listPager.EachListItem(ctx, opt, func(obj runtime.Object) error {
		lease := obj.(*coordinationv1.Lease)
		c.latestLease.Store(lease.Name, lease)
		return nil
	})
}

func (c *NodeLeaseController) syncWorker(ctx context.Context) {
	for nodeName := range c.leaseChan {
		c.sync(ctx, nodeName)
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

// remove removes a lease from the NodeLeaseController
func (c *NodeLeaseController) remove(name string) {
	cancel, ok := c.cancelJob.LoadAndDelete(name)
	if ok {
		cancel()
		c.latestLease.Delete(name)
	}
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
			logger.Error("failed to create lease", err)
			return
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

	logger := log.FromContext(ctx)

	if logger.Enabled(ctx, log.LevelWarn) {
		defer log.Elapsed(ctx, c.clock, log.LevelWarn, time.Second, "Long time to create node lease")()
	}

	lease, err := c.clientSet.CoordinationV1().Leases(c.leaseNamespace).Create(ctx, lease, metav1.CreateOptions{})
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

	logger := log.FromContext(ctx)

	if logger.Enabled(ctx, log.LevelWarn) {
		defer log.Elapsed(ctx, c.clock, log.LevelWarn, time.Second, "Long time to update node lease")()
	}

	lease, err := c.clientSet.CoordinationV1().Leases(c.leaseNamespace).Update(ctx, lease, metav1.UpdateOptions{})
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
