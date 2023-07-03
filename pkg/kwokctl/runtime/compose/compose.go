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

package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/types"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

func (c *Cluster) upCompose(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	conf := &config.Options

	args := []string{"up", "-d"}
	if conf.QuietPull {
		args = append(args, "--quiet-pull")
	}

	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	for i := 0; ctx.Err() == nil; i++ {
		err = c.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
		if err != nil {
			logger.Debug("Failed to start cluster",
				"times", i,
				"err", err,
			)
			time.Sleep(time.Second)
			continue
		}
		ready, err := c.isRunning(ctx)
		if err != nil {
			logger.Debug("Failed to check components status",
				"times", i,
				"err", err,
			)
			time.Sleep(time.Second)
			continue
		}
		if !ready {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	err = ctx.Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) downCompose(ctx context.Context) error {
	args := []string{"down"}
	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	err = c.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) startCompose(ctx context.Context) error {
	// TODO: nerdctl does not support 'compose start' in v1.1.0 or earlier
	// Support in https://github.com/containerd/nerdctl/pull/1656 merge into the main branch, but there is no release
	subcommand := []string{"start"}
	if c.runtime == consts.RuntimeTypeNerdctl {
		subcommand = []string{"up", "-d"}
	}

	commands, err := c.buildComposeCommands(ctx, subcommand...)
	if err != nil {
		return err
	}

	err = c.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		return fmt.Errorf("failed to start cluster: %w", err)
	}

	if c.runtime == consts.RuntimeTypeNerdctl {
		backupFilename := c.GetWorkdirPath("restart.db")
		fi, err := os.Stat(backupFilename)
		if err == nil {
			if fi.IsDir() {
				return fmt.Errorf("wrong backup file %s, it cannot be a directory, please remove it", backupFilename)
			}
			if err := c.SnapshotRestore(ctx, backupFilename); err != nil {
				return fmt.Errorf("failed to restore cluster data: %w", err)
			}
			if err := c.Remove(backupFilename); err != nil {
				return fmt.Errorf("failed to remove backup file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (c *Cluster) stopCompose(ctx context.Context) error {
	// TODO: nerdctl does not support 'compose stop' in v1.0.0 or earlier
	subcommand := "stop"
	if c.runtime == consts.RuntimeTypeNerdctl {
		subcommand = "down"
		err := c.SnapshotSave(ctx, c.GetWorkdirPath("restart.db"))
		if err != nil {
			return fmt.Errorf("failed to snapshot cluster data: %w", err)
		}
	}

	commands, err := c.buildComposeCommands(ctx, subcommand)
	if err != nil {
		return err
	}

	err = c.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}
	return nil
}

// buildComposeCommands returns the compose commands with given current runtime and args
func (c *Cluster) buildComposeCommands(ctx context.Context, args ...string) ([]string, error) {
	if c.composeCommands != nil {
		return append(c.composeCommands, args...), nil
	}
	switch c.runtime {
	case consts.RuntimeTypePodman:
		pc, err := c.preInstallPodmanCompose(ctx)
		if err != nil {
			return nil, err
		}
		c.composeCommands = pc
		return append(pc, args...), nil
	case consts.RuntimeTypeDocker:
		pc, err := c.preInstallDockerCompose(ctx)
		if err != nil {
			return nil, err
		}
		c.composeCommands = pc
		return append(pc, args...), nil
	case consts.RuntimeTypeNerdctl:
		pc := []string{c.runtime, "compose"}
		c.composeCommands = pc
		return append(pc, args...), nil
	default:
		return nil, fmt.Errorf("unknown runtime %s", c.runtime)
	}
}

// preInstallPodmanCompose pre-installs podman-compose
func (c *Cluster) preInstallPodmanCompose(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	p, err := exec.LookPath("podman-compose")
	if err == nil {
		return []string{p}, nil
	}

	pybin, err := exec.LookPath("python3")
	if err != nil {
		pybin, err = exec.LookPath("python")
		if err != nil {
			return nil, fmt.Errorf("failed to find python3 or python")
		}
	}

	target := path.Join(conf.CacheDir, "py", "podman-compose")
	bin := path.Join(target, "podman_compose.py")
	if !file.Exists(bin) {
		err = c.Exec(exec.WithStdIO(ctx), pybin, "-m", "pip", "install", "-t", target, "podman-compose")
		if err != nil {
			return nil, err
		}
		if !file.Exists(bin) {
			return nil, fmt.Errorf("failed to install podman-compose")
		}
	}
	return []string{pybin, bin}, nil
}

// preInstallDockerCompose pre-installs docker-compose
func (c *Cluster) preInstallDockerCompose(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	err = c.Exec(ctx, c.runtime, "compose", "version")
	if err != nil {
		// docker compose subcommand does not exist, try to download it
		dockerComposePath := c.GetBinPath("docker-compose" + conf.BinSuffix)
		err = c.DownloadWithCache(ctx, conf.CacheDir, conf.DockerComposeBinary, dockerComposePath, 0750, conf.QuietPull)
		if err != nil {
			return nil, err
		}
		return []string{dockerComposePath}, nil
	}

	return []string{c.runtime, "compose"}, nil
}

type statusItem struct {
	Service string
	State   string
}

func (c *Cluster) isRunning(ctx context.Context) (bool, error) {
	// podman doesn't support ps --format=json
	if c.runtime == consts.RuntimeTypePodman {
		return true, nil
	}
	commands, err := c.buildComposeCommands(ctx, "ps", "--format=json")
	if err != nil {
		return false, err
	}
	out := bytes.NewBuffer(nil)
	err = c.Exec(exec.WithWriteTo(exec.WithDir(ctx, c.Workdir()), out), commands[0], commands[1:]...)
	if err != nil {
		return false, err
	}

	var data []statusItem
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		return false, err
	}

	if len(data) == 0 {
		logger := log.FromContext(ctx)
		logger.Debug("No components found")
		return false, nil
	}

	components, ok := slices.Find(data, func(i statusItem) bool {
		return i.State != "running"
	})
	if ok {
		logger := log.FromContext(ctx)
		logger.Debug("Components not all running",
			"components", components,
		)
		return false, nil
	}
	return true, nil
}

func convertToCompose(name string, hostIP string, cs []internalversion.Component) *types.Config {
	svcs := convertComponentsToComposeServices(name, hostIP, cs)
	return &types.Config{
		Extensions: map[string]interface{}{
			"version": "3",
		},
		Services: svcs,
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name: name,
			},
		},
	}
}

func convertComponentsToComposeServices(prefix string, hostIP string, cs []internalversion.Component) (svcs types.Services) {
	svcs = make(types.Services, len(cs))
	for i, c := range cs {
		svcs[i] = convertComponentToComposeService(prefix, hostIP, c)
	}
	return svcs
}

func convertComponentToComposeService(prefix string, hostIP string, cs internalversion.Component) (svc types.ServiceConfig) {
	svc.Name = cs.Name
	svc.ContainerName = prefix + "-" + cs.Name
	svc.Image = cs.Image
	svc.Links = cs.Links
	svc.Entrypoint = cs.Command
	if svc.Entrypoint == nil {
		svc.Entrypoint = []string{}
	}
	svc.Command = cs.Args
	svc.Restart = "always"
	svc.Ports = make([]types.ServicePortConfig, len(cs.Ports))
	for i, p := range cs.Ports {
		svc.Ports[i] = types.ServicePortConfig{
			Mode:      "ingress",
			HostIP:    hostIP,
			Target:    p.Port,
			Published: format.String(p.HostPort),
			Protocol:  strings.ToLower(string(p.Protocol)),
		}
	}
	svc.Environment = make(map[string]*string, len(cs.Envs))
	for i, e := range cs.Envs {
		svc.Environment[e.Name] = &cs.Envs[i].Value
	}
	svc.Volumes = make([]types.ServiceVolumeConfig, len(cs.Volumes))
	for i, v := range cs.Volumes {
		svc.Volumes[i] = types.ServiceVolumeConfig{
			Type:     types.VolumeTypeBind,
			Source:   v.HostPath,
			Target:   v.MountPath,
			ReadOnly: v.ReadOnly,
		}
	}
	return svc
}
