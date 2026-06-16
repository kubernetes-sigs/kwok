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

package kind

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	logger := log.FromContext(ctx)

	if !slices.Contains(config.Options.Components, consts.ComponentKubeScheduler) {
		defer func() {
			err := c.StopComponent(ctx, consts.ComponentKubeScheduler)
			if err != nil {
				logger.Error("Failed to disable kube-scheduler",
					"err", err,
				)
			}
		}()
	}

	if !slices.Contains(config.Options.Components, consts.ComponentKubeControllerManager) {
		defer func() {
			err := c.StopComponent(ctx, consts.ComponentKubeControllerManager)
			if err != nil {
				logger.Error("Failed to disable kube-controller-manager",
					"err", err,
				)
			}
		}()
	}

	err = c.SetConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// This needs to be done before starting the cluster
	err = c.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	images, err := c.listAllImages(ctx)
	if err != nil {
		return err
	}

	args := []string{
		"create", "cluster",
		"--config", c.GetWorkdirPath(runtime.KindName),
		"--name", c.Name(),
		"--image", conf.KindNodeImage,
	}

	deadline, ok := ctx.Deadline()
	if ok {
		wait := time.Until(deadline)
		if wait < 0 {
			wait = time.Minute
		}
		args = append(args, "--wait", format.HumanDuration(wait))
	} else {
		args = append(args, "--wait", "1m")
	}

	err = c.Exec(utilsexec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath, args...)
	if err != nil {
		return err
	}

	err = c.loadImages(ctx, kindPath, images, conf.CacheDir)
	if err != nil {
		return err
	}

	// TODO: remove this when kind support set server
	err = c.fillKubeconfigContextServer(conf.BindAddress)
	if err != nil {
		return err
	}

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = c.Kubectl(utilsexec.WithWriteTo(ctx, kubeconfigBuf), "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = c.WriteFile(kubeconfigPath, kubeconfigBuf.Bytes())
	if err != nil {
		return err
	}

	// Cordoning the node to prevent fake pods from being scheduled on it
	err = c.Kubectl(ctx, "cordon", c.getClusterName())
	if err != nil {
		logger.Error("Failed cordon node",
			"err", err,
		)
	}

	err = c.Exec(ctx, c.runtime, "exec", c.getClusterName(), "chmod", "-R", "+r", "/etc/kubernetes/pki")
	if err != nil {
		logger.Error("Failed to chmod pki",
			"err", err,
		)
	}

	return nil
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	err = c.Exec(utilsexec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath, "delete", "cluster", "--name", c.Name())
	if err != nil {
		logger.Error("Failed to delete cluster",
			"err", err,
		)
	}

	return nil
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	err := c.Exec(ctx, c.runtime, "start", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	err := c.Exec(ctx, c.runtime, "stop", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

var startImportantComponents = map[string]struct{}{
	consts.ComponentEtcd: {},
}

var stopImportantComponents = map[string]struct{}{
	consts.ComponentEtcd:          {},
	consts.ComponentKubeApiserver: {},
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"component", name,
	)
	if _, important := startImportantComponents[name]; !important {
		if !c.IsDryRun() {
			if _, _, exist, err := c.inspectComponent(ctx, name); err != nil {
				return err
			} else if exist {
				logger.Debug("Component already started")
				return nil
			}
		}
	}

	logger.Debug("Starting component")
	err := c.Exec(ctx, c.runtime, "exec", c.getClusterName(), "mv", "/etc/kubernetes/"+name+".yaml.bak", "/etc/kubernetes/manifests/"+name+".yaml")
	if err != nil {
		return err
	}
	if _, important := startImportantComponents[name]; important {
		return nil
	}
	if c.IsDryRun() {
		return nil
	}
	return c.waitComponentReady(ctx, name, true, 120*time.Second)
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"component", name,
	)
	if _, important := stopImportantComponents[name]; !important {
		if !c.IsDryRun() {
			if _, _, exist, err := c.inspectComponent(ctx, name); err != nil {
				return err
			} else if !exist {
				logger.Debug("Component already stopped")
				return nil
			}
		}
	}

	logger.Debug("Stopping component")
	err := c.Exec(ctx, c.runtime, "exec", c.getClusterName(), "mv", "/etc/kubernetes/manifests/"+name+".yaml", "/etc/kubernetes/"+name+".yaml.bak")
	if err != nil {
		return err
	}
	// Once etcd and kube-apiserver are stopped, the cluster will go down
	if _, important := stopImportantComponents[name]; important {
		return nil
	}
	if c.IsDryRun() {
		return nil
	}
	return c.waitComponentReady(ctx, name, false, 120*time.Second)
}

func (c *Cluster) listAllImages(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	images := []string{}
	for _, component := range config.Components {
		if component.Image == "" {
			continue
		}
		images = append(images, component.Image)
	}

	return images, nil
}

// loadDockerImages loads docker images into the cluster.
// `kind load docker-image`
func (c *Cluster) loadDockerImages(ctx context.Context, command string, kindCluster string, images []string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		err := c.Exec(c.withProviderEnv(ctx),
			command, "load", "docker-image",
			image,
			"--name", kindCluster,
		)
		if err != nil {
			return err
		}
		logger.Info("Loaded image",
			"image", image,
		)
	}
	return nil
}

// loadArchiveImages loads docker images into the cluster.
// `kind load image-archive`
func (c *Cluster) loadArchiveImages(ctx context.Context, command string, kindCluster string, images []string, runtime string, tmpDir string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		archive := utilspath.Join(tmpDir, "image-archive", strings.ReplaceAll(image, ":", "/")+".tar")
		err := c.MkdirAll(filepath.Dir(archive))
		if err != nil {
			return err
		}

		err = c.Exec(ctx, runtime, "save", image, "-o", archive)
		if err != nil {
			return err
		}
		err = c.Exec(c.withProviderEnv(ctx),
			command, "load", "image-archive",
			archive,
			"--name", kindCluster,
		)
		if err != nil {
			return err
		}
		logger.Info("Loaded image",
			"image", image,
		)
		err = c.Remove(archive)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) loadImages(ctx context.Context, kindPath string, images []string, cacheDir string) error {
	var err error
	if c.runtime == consts.RuntimeTypeDocker {
		err = c.loadDockerImages(ctx, kindPath, c.Name(), images)
	} else {
		err = c.loadArchiveImages(ctx, kindPath, c.Name(), images, c.runtime, cacheDir)
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) getClusterName() string {
	return c.Name() + "-control-plane"
}

func (c *Cluster) getComponentName(name string) string {
	clusterName := c.getClusterName()
	return name + "-" + clusterName
}
