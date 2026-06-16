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
	"fmt"
	"os/user"
	"strconv"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func (c *Cluster) startComponent(ctx context.Context, component internalversion.Component) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"component", component.Name,
	)
	if c.isRunning(ctx, component) {
		logger.Debug("Component already started")
		return nil
	}

	if len(component.Envs) > 0 {
		ctx = utilsexec.WithEnv(ctx, utilsslices.Map(component.Envs, func(c internalversion.Env) string {
			return fmt.Sprintf("%s=%s", c.Name, c.Value)
		}))
	}

	if component.User != "" {
		u, err := user.Lookup(component.User)
		if err != nil {
			return err
		}
		uid, err := strconv.ParseInt(u.Uid, 0, 64)
		if err != nil {
			return err
		}
		gid, err := strconv.ParseInt(u.Gid, 0, 64)
		if err != nil {
			return err
		}
		ctx = utilsexec.WithUser(ctx, &uid, &gid)
	}

	logger.Debug("Starting component")
	return c.ForkExec(ctx, component.WorkDir, component.Binary, component.Args...)
}

func (c *Cluster) startComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, false, true, func(ctx context.Context, component internalversion.Component) error {
		return c.startComponent(ctx, component)
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) stopComponent(ctx context.Context, component internalversion.Component) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"component", component.Name,
	)
	if !c.isRunning(ctx, component) {
		logger.Debug("Component already stopped")
		return nil
	}
	logger.Debug("Stopping component")
	return c.ForkExecKill(ctx, component.WorkDir, component.Binary)
}

func (c *Cluster) stopComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, true, true, func(ctx context.Context, component internalversion.Component) error {
		return c.stopComponent(ctx, component)
	})
	if err != nil {
		return err
	}
	return nil
}

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	return c.start(ctx)
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	return c.stop(ctx)
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	return c.start(ctx)
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	return c.stop(ctx)
}

func (c *Cluster) start(ctx context.Context) error {
	err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := c.startComponents(ctx)
		return err == nil, err
	},
		wait.WithContinueOnError(5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}

	if !c.IsDryRun() {
		logger := log.FromContext(ctx)
		err = c.waitServed(ctx, 2*time.Minute)
		if err != nil {
			logger.Warn("Cluster is not served yet",
				"err", err,
			)
		}
	}
	return nil
}

func (c *Cluster) served(ctx context.Context) (bool, error) {
	err := c.KubectlInCluster(ctx, "get", "--raw", "/version")
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cluster) waitServed(ctx context.Context, timeout time.Duration) error {
	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.served(ctx)
		if err != nil {
			logger.Debug("Cluster is not served yet",
				"err", err,
			)
		}
		return ready, nil
	},
		wait.WithTimeout(timeout),
		wait.WithInterval(time.Second/5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func (c *Cluster) stop(ctx context.Context) error {
	err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := c.stopComponents(ctx)
		return err == nil, err
	},
		wait.WithContinueOnError(5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}

	return nil
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	component, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	err = c.startComponent(ctx, component)
	if err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}
	return nil
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	component, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	err = c.stopComponent(ctx, component)
	if err != nil {
		return fmt.Errorf("failed to stop %s: %w", name, err)
	}
	return nil
}
