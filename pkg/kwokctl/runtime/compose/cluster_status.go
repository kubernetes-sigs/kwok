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

package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
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
	running, _ := c.inspectComponent(ctx, name)

	if !running {
		return runtime.ComponentStatusStopped, nil
	}

	// TODO: check if the component is ready

	return runtime.ComponentStatusReady, nil
}

func (c *Cluster) inspectComponent(ctx context.Context, componentName string) (running bool, exist bool) {
	buf := bytes.NewBuffer(nil)
	args := make([]string, 0, 3)
	args = append(args, "inspect", c.Name()+"-"+componentName)

	args = append(args, "--format={{ json . }}")

	err := c.Exec(utilsexec.WithWriteTo(ctx, buf), c.runtime, args...)
	if err != nil {
		// TODO: check if component exists or other error
		return false, false
	}

	running, err = checkInspect(buf.Bytes())
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Warn("Failed to check inspect result",
			"err", err,
		)
		return false, false
	}
	return running, true
}

type inspectStatus struct {
	State struct {
		Running bool
	}
}

func checkInspect(raw []byte) (bool, error) {
	if len(raw) == 0 {
		return false, fmt.Errorf("empty inspect result")
	}
	raw = bytes.TrimSpace(raw)
	switch raw[0] {
	case '{':
		var tmp inspectStatus

		err := json.Unmarshal(raw, &tmp)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal inspect result: %w", err)
		}

		return tmp.State.Running, nil
	case '[':
		var tmp []inspectStatus
		err := json.Unmarshal(raw, &tmp)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal inspect result: %w", err)
		}

		if len(tmp) == 0 {
			return false, nil
		}
		return tmp[0].State.Running, nil
	default:
		return false, fmt.Errorf("unexpected inspect result: %s", raw)
	}
}
