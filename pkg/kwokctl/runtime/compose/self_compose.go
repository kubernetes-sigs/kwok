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

package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func (c *Cluster) networkName() string {
	return c.Name()
}

func (c *Cluster) createNetwork(ctx context.Context) error {
	network := c.networkName()
	logger := log.FromContext(ctx)
	logger = logger.With("network", network)

	if !c.IsDryRun() {
		if exist := c.inspectNetwork(ctx, network); exist {
			logger.Debug("Network already exists")
			return nil
		}
	}
	args := []string{
		"network", "create", network,
	}
	if !c.isAppleContainer {
		args = append(args, c.labelArgs()...)
	}
	logger.Debug("Creating network")
	return c.Exec(ctx, c.runtime, args...)
}

func (c *Cluster) deleteNetwork(ctx context.Context) error {
	network := c.networkName()
	logger := log.FromContext(ctx)
	logger = logger.With("network", network)
	if !c.IsDryRun() {
		if exist := c.inspectNetwork(ctx, network); !exist {
			logger.Debug("Network does not exist")
			return nil
		}
	}
	args := []string{
		"network", "rm", network,
	}
	logger.Debug("Deleting network")
	err := c.Exec(ctx, c.runtime, args...)
	if err != nil {
		if c.isAppleContainer {
			return nil
		}

		if !c.isNerdctl {
			return err
		}

		errMessage := err.Error()
		if !strings.Contains(errMessage, "is in use by container") {
			return err
		}

		logger.Warn("Network is in use by container, try to delete containers")
		if err := c.deleteComponents(ctx); err != nil {
			return err
		}

		err = wait.Poll(ctx,
			func(ctx context.Context) (bool, error) {
				if exist := c.inspectNetwork(ctx, network); !exist {
					return true, nil
				}
				logger.Warn("Retrying to delete network")
				err := c.Exec(ctx, c.runtime, args...)
				return err == nil, err
			},
			wait.WithContinueOnError(2),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) inspectNetwork(ctx context.Context, name string) (exist bool) {
	if c.isAppleContainer {
		buf := bytes.NewBuffer(nil)
		ctx := exec.WithIOStreams(ctx, exec.IOStreams{
			Out: buf,
		})
		err := c.Exec(ctx, c.runtime, "network", "inspect", name)
		if err != nil {
			// TODO: check if network exists or other error
			return false
		}
		if strings.TrimSpace(buf.String()) == "[]" {
			return false
		}

		return true
	}
	err := c.Exec(ctx, c.runtime, "network", "inspect", name)
	if err != nil {
		// TODO: check if network exists or other error
		return false
	}
	return true
}

// On Nerdctl, need to check if --restart=unless-stopped is supported
// https://github.com/containerd/containerd/pull/6744
func (c *Cluster) isCanNerdctlUnlessStopped(ctx context.Context) (bool, error) {
	if !c.isNerdctl {
		return false, fmt.Errorf("canNerdctlUnlessStopped only for nerdctl")
	}

	if c.IsDryRun() {
		return true, nil
	}

	if c.canNerdctlUnlessStopped != nil {
		return *c.canNerdctlUnlessStopped, nil
	}

	var canNerdctlUnlessStopped *bool
	logger := log.FromContext(ctx)
	nerdctlVersion, err := c.ParseVersionFromBinary(ctx, c.runtime)
	if err != nil {
		logger.Warn("Failed to parse nerdctl version", "err", err)
	} else if nerdctlVersion.LE(version.NewVersion(1, 3, 0)) {
		canNerdctlUnlessStopped = format.Ptr(false)
		logger = logger.With("nerdctlCheck", nerdctlVersion)
	}

	if canNerdctlUnlessStopped == nil {
		buf := bytes.NewBuffer(nil)
		err = c.Exec(exec.WithWriteTo(ctx, buf), c.runtime, "create", "--help")
		if err != nil {
			return false, fmt.Errorf("canNerdctlUnlessStopped failed: %w", err)
		}
		logger = logger.With("containerdCheck", "not support unless-stopped")
		canNerdctlUnlessStopped = format.Ptr(strings.Contains(buf.String(), "unless-stopped"))
	}

	if !*canNerdctlUnlessStopped {
		// https://github.com/containerd/containerd/pull/6744
		// Nerdctl unless-stopped is depends on containerd
		logger.Warn("nerdctl or containerd version is too low, " +
			"suggested upgrade nerdctl and containerd for a better experience",
		)
	}

	c.canNerdctlUnlessStopped = canNerdctlUnlessStopped
	return *canNerdctlUnlessStopped, nil
}

func (c *Cluster) labelArgs() []string {
	args := []string{}
	switch c.runtime {
	case consts.RuntimeTypeDocker:
		args = append(args, "--label=com.docker.compose.project="+c.Name())
	case consts.RuntimeTypePodman:
		// https://github.com/containers/podman-compose/blob/f6dbce36181c44d0d08b6f4ca166508542875ce1/podman_compose.py#L729
		args = append(args, "--label=io.podman.compose.project="+c.Name())
		args = append(args, "--label=com.docker.compose.project="+c.Name())
	default:
		if c.isNerdctl {
			// https://github.com/containerd/nerdctl/blob/3c9300207f45c4a0422d8381d58c5be06bb49b39/pkg/labels/labels.go#L33
			args = append(args, "--label=com.docker.compose.project="+c.Name())
		}
	}
	return args
}

func (c *Cluster) createComponent(ctx context.Context, componentName string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", componentName)
	if !c.IsDryRun() {
		if _, exist := c.inspectComponent(ctx, componentName); exist {
			logger.Debug("Component already exists")
			return nil
		}
	}
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	component, ok := slices.Find(conf.Components, func(component internalversion.Component) bool {
		return component.Name == componentName
	})
	if !ok {
		return fmt.Errorf("component %s not found", componentName)
	}

	args := []string{"create",
		"--name=" + c.Name() + "-" + componentName,
	}

	if !c.isAppleContainer {
		args = append(args, "--pull=never")
	}

	entrypoint := strings.Join(component.Command, " ")
	if entrypoint != "" {
		args = append(args, "--entrypoint="+entrypoint)
	}

	network := c.networkName()
	if network != "" {
		args = append(args, "--network="+network)
	}

	if component.User != "" {
		args = append(args, "--user="+component.User)
	}

	switch c.runtime {
	case consts.RuntimeTypeDocker:
		for _, link := range component.Links {
			args = append(args, "--link="+c.Name()+"-"+link)
		}
	case consts.RuntimeTypePodman:
		for _, link := range component.Links {
			args = append(args, "--requires="+c.Name()+"-"+link)
		}
	default:
		// Nerdctl does not support --link and --requires
	}

	switch c.runtime {
	case consts.RuntimeTypeDocker, consts.RuntimeTypePodman:
		args = append(args, "--restart=unless-stopped")
	default:
		if c.isNerdctl {
			canNerdctlUnlessStopped, err := c.isCanNerdctlUnlessStopped(ctx)
			if err != nil {
				logger.Error("Failed to check unless-stopped support", err)
			}
			if canNerdctlUnlessStopped {
				args = append(args, "--restart=unless-stopped")
			} else {
				args = append(args, "--restart=always")
			}
		}
	}

	args = append(args, c.labelArgs()...)

	for _, port := range component.Ports {
		if port.HostPort == 0 {
			continue
		}
		protocol := port.Protocol
		if protocol == "" {
			protocol = internalversion.ProtocolTCP
		}
		args = append(args, "--publish="+format.String(port.HostPort)+":"+format.String(port.Port)+"/"+strings.ToLower(string(protocol)))
	}
	for _, volume := range component.Volumes {
		if volume.ReadOnly {
			args = append(args, "--volume="+volume.HostPath+":"+volume.MountPath+":ro")
		} else {
			args = append(args, "--volume="+volume.HostPath+":"+volume.MountPath)
		}
	}
	for _, env := range component.Envs {
		args = append(args, "--env="+env.Name+"="+env.Value)
	}

	args = append(args, component.Image)
	args = append(args, component.Args...)

	logger.Debug("Creating component")
	return c.Exec(ctx, c.runtime, args...)
}

func (c *Cluster) createComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, false, true, func(ctx context.Context, component internalversion.Component) error {
		return c.createComponent(ctx, component.Name)
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) deleteComponent(ctx context.Context, componentName string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", componentName)
	if !c.IsDryRun() {
		if running, exist := c.inspectComponent(ctx, componentName); !exist {
			logger.Debug("Component does not exist")
			return nil
		} else if running {
			if c.isNerdctl {
				// TODO: Remove this after nerdctl fix
				// https://github.com/containerd/nerdctl/issues/1980
				if canNerdctlUnlessStopped, _ := c.isCanNerdctlUnlessStopped(ctx); canNerdctlUnlessStopped {
					return fmt.Errorf("component %s is running, need to stop it first", componentName)
				}
			} else {
				return fmt.Errorf("component %s is running, need to stop it first", componentName)
			}
		}
	}

	args := []string{"rm",
		c.Name() + "-" + componentName,
		"--force",
	}

	logger.Debug("Deleting component")
	return c.Exec(ctx, c.runtime, args...)
}

func (c *Cluster) deleteComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, true, true, func(ctx context.Context, component internalversion.Component) error {
		return c.deleteComponent(ctx, component.Name)
	})
	if err != nil {
		return err
	}
	return nil
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

type inspectAppleContaeinerStatus struct {
	Status string `json:"status"`
}

func checkAppleInspect(raw []byte) (running bool, exist bool, err error) {
	var tmp []inspectAppleContaeinerStatus
	err = json.Unmarshal(raw, &tmp)
	if err != nil {
		return false, false, fmt.Errorf("failed to unmarshal inspect result: %w", err)
	}

	if len(tmp) == 0 {
		return false, false, nil
	}
	return tmp[0].Status == "running", true, nil
}

func (c *Cluster) inspectComponent(ctx context.Context, componentName string) (running bool, exist bool) {
	buf := bytes.NewBuffer(nil)
	args := []string{"inspect", c.Name() + "-" + componentName}

	if !c.isAppleContainer {
		args = append(args, "--format={{ json . }}")
	}

	err := c.Exec(exec.WithWriteTo(ctx, buf), c.runtime, args...)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Debug("Failed to inspect component", "component", componentName, "err", err)
		return false, false
	}

	if c.isAppleContainer {
		running, exist, err = checkAppleInspect(buf.Bytes())
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Warn("Failed to check inspect result", "err", err)
			return false, false
		}
		return running, exist
	}

	running, err = checkInspect(buf.Bytes())
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Warn("Failed to check inspect result", "err", err)
		return false, false
	}
	return running, true
}

func (c *Cluster) startComponent(ctx context.Context, componentName string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", componentName)
	if !c.IsDryRun() {
		if running, exist := c.inspectComponent(ctx, componentName); !exist {
			return fmt.Errorf("component %s does not exist", componentName)
		} else if running {
			logger.Debug("Component already started")
			return nil
		}
	}

	args := []string{
		"start",
		c.Name() + "-" + componentName,
	}

	logger.Debug("Starting component")
	err := c.Exec(ctx, c.runtime, args...)
	if err != nil {
		// TODO: Remove this after nerdctl fix
		// https://github.com/containerd/nerdctl/issues/2270
		if c.isNerdctl {
			errMessage := err.Error()
			switch {
			case strings.Contains(errMessage, "already exists"),
				strings.Contains(errMessage, "file exists"):
				logger.Warn("Component may already started, ignore error, "+
					"see https://github.com/containerd/nerdctl/issues/2270",
					"err", errMessage,
				)
			case strings.Contains(errMessage, "cannot start a container that has stopped"):
				logger.Warn("Component stopped, nerdctl create will start containers, ignore error, "+
					"see https://github.com/containerd/nerdctl/issues/2270",
					"err", errMessage,
				)
			default:
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (c *Cluster) startComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, false, true, func(ctx context.Context, component internalversion.Component) error {
		return c.startComponent(ctx, component.Name)
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) stopComponent(ctx context.Context, componentName string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", componentName)
	if !c.IsDryRun() {
		if running, exist := c.inspectComponent(ctx, componentName); !exist {
			logger.Debug("Component does not exist")
			return nil
		} else if !running {
			logger.Debug("Component already stopped")
			return nil
		}
	}

	args := []string{"stop",
		c.Name() + "-" + componentName,
		"--time=0",
	}

	logger.Debug("Stopping component")
	return c.Exec(ctx, c.runtime, args...)
}

func (c *Cluster) stopComponents(ctx context.Context) error {
	err := c.ForeachComponents(ctx, true, false, func(ctx context.Context, component internalversion.Component) error {
		return c.stopComponent(ctx, component.Name)
	})
	if err != nil {
		return err
	}
	return nil
}
