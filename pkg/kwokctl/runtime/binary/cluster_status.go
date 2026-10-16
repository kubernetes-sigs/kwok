/*
Copyright 2026 The Kubernetes Authors.

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

package binary

import (
	"context"
	"net"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	// TODO: Only the necessary components are checked for readiness.
	for _, component := range config.Components {
		s, _ := c.InspectComponent(ctx, component.Name)
		if s != runtime.ComponentStatusReady {
			return false, nil
		}
	}

	return c.Cluster.Ready(ctx)
}

// WaitReady waits for the cluster to be ready.
func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	if c.IsDryRun() {
		return nil
	}

	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.Ready(ctx)
		if err != nil {
			logger.Debug("Cluster is not ready",
				"err", err,
			)
		}
		return ready, nil
	},
		wait.WithTimeout(timeout),
		wait.WithContinueOnError(10),
		wait.WithInterval(time.Second/2),
	)
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// InspectComponent returns the status of the component
func (c *Cluster) InspectComponent(ctx context.Context, name string) (runtime.ComponentStatus, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return runtime.ComponentStatusUnknown, err
	}
	component, err := c.GetComponent(ctx, name)
	if err != nil {
		return runtime.ComponentStatusUnknown, err
	}

	running := c.isRunning(ctx, component)
	host := probeHost(config.Options.BindAddress)
	return runtime.ComponentStatusWithProbe(ctx, running, component.ReadinessProbe, host, runtime.DialProbeRoundTrip(host)), nil
}

// probeHost returns the address used to reach a component listening on bindAddress.
// Components bound to all interfaces are reached over loopback.
func probeHost(bindAddress string) string {
	ip := net.ParseIP(bindAddress)
	switch {
	case bindAddress == "", ip != nil && ip.IsUnspecified() && ip.To4() != nil:
		return utilsnet.LocalAddress
	case ip != nil && ip.IsUnspecified():
		return net.IPv6loopback.String()
	}
	return bindAddress
}

func (c *Cluster) isRunning(ctx context.Context, component internalversion.Component) bool {
	return c.ForkExecIsRunning(ctx, component.WorkDir, component.Binary)
}
