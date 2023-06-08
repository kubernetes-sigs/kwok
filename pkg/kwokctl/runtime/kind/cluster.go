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

package kind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/image"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// Cluster is an implementation of Runtime for kind
type Cluster struct {
	*runtime.Cluster

	runtime string
}

// NewDockerCluster creates a new Runtime for kind with docker
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
}

// NewPodmanCluster creates a new Runtime for kind with podman
func NewPodmanCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypePodman,
	}, nil
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	return exec.Exec(ctx, c.runtime, "version")
}

// https://github.com/kubernetes-sigs/kind/blob/7b017b2ce14a7fdea9d3ed2fa259c38c927e2dd1/pkg/internal/runtime/runtime.go
func (c *Cluster) withProviderEnv(ctx context.Context) context.Context {
	provider := ""
	if c.runtime == consts.RuntimeTypePodman {
		provider = c.runtime
	}
	ctx = exec.WithEnv(ctx, []string{
		"KIND_EXPERIMENTAL_PROVIDER=" + provider,
	})
	return ctx
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	logger := log.FromContext(ctx)
	verbosity := logger.Level()
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	inClusterKubeconfig := "/etc/kubernetes/scheduler.conf"

	var featureGates []string
	var runtimeConfig []string
	if conf.KubeFeatureGates != "" {
		featureGates = strings.Split(strings.ReplaceAll(conf.KubeFeatureGates, "=", ": "), ",")
	}
	if conf.KubeRuntimeConfig != "" {
		runtimeConfig = strings.Split(strings.ReplaceAll(conf.KubeRuntimeConfig, "=", ": "), ",")
	}

	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		err = file.Create(auditLogPath, 0640)
		if err != nil {
			return err
		}

		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
		err = file.Copy(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	schedulerConfigPath := ""
	if !conf.DisableKubeScheduler && conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = k8s.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, inClusterKubeconfig)
		if err != nil {
			return err
		}
	}

	configPath := c.GetWorkdirPath(runtime.ConfigName)

	etcdComponentPatches := runtime.GetComponentPatches(config, "etcd")
	kubeApiserverComponentPatches := runtime.GetComponentPatches(config, "kube-apiserver")
	kubeSchedulerComponentPatches := runtime.GetComponentPatches(config, "kube-scheduler")
	kubeControllerManagerComponentPatches := runtime.GetComponentPatches(config, "kube-controller-manager")
	kwokControllerComponentPatches := runtime.GetComponentPatches(config, "kwok-controller")
	extraLogVolumes := runtime.GetLogVolumes(ctx)
	kwokControllerExtraVolumes := kwokControllerComponentPatches.ExtraVolumes
	kwokControllerExtraVolumes = append(kwokControllerExtraVolumes, extraLogVolumes...)
	kindYaml, err := BuildKind(BuildKindConfig{
		BindAddress:                   conf.BindAddress,
		KubeApiserverPort:             conf.KubeApiserverPort,
		EtcdPort:                      conf.EtcdPort,
		PrometheusPort:                conf.PrometheusPort,
		KwokControllerPort:            conf.KwokControllerPort,
		FeatureGates:                  featureGates,
		RuntimeConfig:                 runtimeConfig,
		AuditPolicy:                   auditPolicyPath,
		AuditLog:                      auditLogPath,
		SchedulerConfig:               schedulerConfigPath,
		ConfigPath:                    configPath,
		Verbosity:                     verbosity,
		EtcdExtraArgs:                 etcdComponentPatches.ExtraArgs,
		EtcdExtraVolumes:              etcdComponentPatches.ExtraVolumes,
		ApiserverExtraArgs:            kubeApiserverComponentPatches.ExtraArgs,
		ApiserverExtraVolumes:         kubeApiserverComponentPatches.ExtraVolumes,
		SchedulerExtraArgs:            kubeSchedulerComponentPatches.ExtraArgs,
		SchedulerExtraVolumes:         kubeSchedulerComponentPatches.ExtraVolumes,
		ControllerManagerExtraArgs:    kubeControllerManagerComponentPatches.ExtraArgs,
		ControllerManagerExtraVolumes: kubeControllerManagerComponentPatches.ExtraVolumes,
		KwokControllerExtraVolumes:    kwokControllerExtraVolumes,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(c.GetWorkdirPath(runtime.KindName), []byte(kindYaml), 0640)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	kwokControllerPod, err := BuildKwokControllerPod(BuildKwokControllerPodConfig{
		KwokControllerImage:      conf.KwokControllerImage,
		Name:                     c.Name(),
		Verbosity:                verbosity,
		NodeLeaseDurationSeconds: 40,
		ExtraArgs:                kwokControllerComponentPatches.ExtraArgs,
		ExtraVolumes:             kwokControllerExtraVolumes,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(c.GetWorkdirPath(runtime.KwokPod), []byte(kwokControllerPod), 0640)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KwokPod, err)
	}

	if conf.PrometheusPort != 0 {
		prometheusPatches := runtime.GetComponentPatches(config, "prometheus")
		prometheusConf := BuildPrometheusDeploymentConfig{
			PrometheusImage: conf.PrometheusImage,
			Name:            c.Name(),
			ExtraArgs:       prometheusPatches.ExtraArgs,
			ExtraVolumes:    prometheusPatches.ExtraVolumes,
		}
		if verbosity != log.LevelInfo {
			prometheusConf.LogLevel = log.ToLogSeverityLevel(verbosity)
		}
		prometheusDeploy, err := BuildPrometheusDeployment(prometheusConf)
		if err != nil {
			return err
		}
		err = os.WriteFile(c.GetWorkdirPath(runtime.PrometheusDeploy), []byte(prometheusDeploy), 0640)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.PrometheusDeploy, err)
		}
	}

	images := []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	err = image.PullImages(ctx, c.runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}

	return nil
}

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	ctx = c.withProviderEnv(ctx)

	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	config.Components = append(config.Components,
		internalversion.Component{
			Name: "etcd",
		},
		internalversion.Component{
			Name: "kube-apiserver",
		},
		internalversion.Component{
			Name: "kwok-controller",
		},
	)

	if conf.PrometheusPort != 0 {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: "prometheus",
			},
		)
	}

	if !conf.DisableKubeScheduler {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: "kube-scheduler",
			},
		)
	}

	if !conf.DisableKubeControllerManager {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: "kube-controller-manager",
			},
		)
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

	logger := log.FromContext(ctx)

	kindPath, err := c.preDownloadKind(ctx)
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

	err = exec.Exec(exec.WithAllWriteToErrOut(ctx), kindPath, args...)
	if err != nil {
		return err
	}

	// TODO: remove this when kind support set server
	err = c.fillKubeconfigContextServer(conf.BindAddress)
	if err != nil {
		return err
	}

	images := []string{conf.KwokControllerImage}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}

	if c.runtime == consts.RuntimeTypeDocker {
		err = loadDockerImages(ctx, kindPath, c.Name(), images)
	} else {
		err = loadArchiveImages(ctx, kindPath, c.Name(), images, c.runtime, conf.CacheDir)
	}
	if err != nil {
		return err
	}

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = c.Kubectl(exec.WithWriteTo(ctx, kubeconfigBuf), "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = os.WriteFile(kubeconfigPath, kubeconfigBuf.Bytes(), 0640)
	if err != nil {
		return err
	}

	err = exec.Exec(ctx, c.runtime, "cp", c.GetWorkdirPath(runtime.KwokPod), c.Name()+"-control-plane:/etc/kubernetes/manifests/kwok-controller.yaml")
	if err != nil {
		return err
	}

	if conf.PrometheusPort != 0 {
		err = c.Kubectl(exec.WithAllWriteToErrOut(ctx), "apply", "-f", c.GetWorkdirPath(runtime.PrometheusDeploy))
		if err != nil {
			return err
		}
	}

	err = c.Kubectl(ctx, "cordon", c.getClusterName())
	if err != nil {
		logger.Error("Failed cordon node", err)
	}

	if conf.DisableKubeScheduler {
		err := c.StopComponent(ctx, "kube-scheduler")
		if err != nil {
			logger.Error("Failed to disable kube-scheduler", err)
		}
	}

	if conf.DisableKubeControllerManager {
		err := c.StopComponent(ctx, "kube-controller-manager")
		if err != nil {
			logger.Error("Failed to disable kube-controller-manager", err)
		}
	}

	return nil
}

// loadDockerImages loads docker images into the cluster.
// `kind load docker-image`
func loadDockerImages(ctx context.Context, command string, kindCluster string, images []string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		err := exec.Exec(ctx,
			command, "load", "docker-image",
			image,
			"--name", kindCluster,
		)
		if err != nil {
			return err
		}
		logger.Info("Loaded image", "image", image)
	}
	return nil
}

// loadArchiveImages loads docker images into the cluster.
// `kind load image-archive`
func loadArchiveImages(ctx context.Context, command string, kindCluster string, images []string, runtime string, tmpDir string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		archive := path.Join(tmpDir, "image-archive", strings.ReplaceAll(image, ":", "/")+".tar")
		err := os.MkdirAll(filepath.Dir(archive), 0750)
		if err != nil {
			return err
		}

		err = exec.Exec(ctx, runtime, "save", image, "-o", archive)
		if err != nil {
			return err
		}
		err = exec.Exec(ctx,
			command, "load", "image-archive",
			archive,
			"--name", kindCluster,
		)
		if err != nil {
			return err
		}
		logger.Info("Loaded image", "image", image)
		err = file.Remove(archive)
		if err != nil {
			return err
		}
	}
	return nil
}

// WaitReady waits for the cluster to be ready.
func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
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
	}, wait.WithTimeout(timeout), wait.WithImmediate())
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	ok, err := c.Cluster.Ready(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	out := bytes.NewBuffer(nil)
	err = c.KubectlInCluster(exec.WithWriteTo(ctx, out), "get", "pod", "--namespace=kube-system", "--field-selector=status.phase!=Running", "--output=json")
	if err != nil {
		return false, err
	}

	var data corev1.PodList
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		return false, err
	}

	if len(data.Items) != 0 {
		logger := log.FromContext(ctx)
		logger.Debug("Components not all running",
			"components", slices.Map(data.Items, func(item corev1.Pod) interface{} {
				return struct {
					Pod   string
					Phase string
				}{
					Pod:   log.KObj(&item).String(),
					Phase: string(item.Status.Phase),
				}
			}),
		)
		return false, nil
	}
	return true, nil
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	ctx = c.withProviderEnv(ctx)

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	err = exec.Exec(exec.WithAllWriteToErrOut(ctx), kindPath, "delete", "cluster", "--name", c.Name())
	if err != nil {
		logger.Error("Failed to delete cluster", err)
	}

	return nil
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	err := exec.Exec(ctx, c.runtime, "start", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	err := exec.Exec(ctx, c.runtime, "stop", c.getClusterName())
	if err != nil {
		return err
	}
	return nil
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	err := exec.Exec(ctx, c.runtime, "exec", c.getClusterName(), "mv", "/etc/kubernetes/"+name+".yaml.bak", "/etc/kubernetes/manifests/"+name+".yaml")
	if err != nil {
		return err
	}

	return c.waitComponentReady(ctx, name, true, 120*time.Second)
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	err := exec.Exec(ctx, c.runtime, "exec", c.getClusterName(), "mv", "/etc/kubernetes/manifests/"+name+".yaml", "/etc/kubernetes/"+name+".yaml.bak")
	if err != nil {
		return err
	}

	if name == "etcd" || name == "kube-apiserver" {
		return nil
	}
	return c.waitComponentReady(ctx, name, false, 120*time.Second)
}

// waitComponentReady waits for a component to be ready
func (c *Cluster) waitComponentReady(ctx context.Context, name string, wantReady bool, timeout time.Duration) error {
	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.componentReady(ctx, name)
		if err != nil {
			logger.Debug("check component ready",
				"component", name,
				"err", err,
			)
		}
		return ready == wantReady, nil
	}, wait.WithTimeout(timeout), wait.WithImmediate())
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func (c *Cluster) componentReady(ctx context.Context, name string) (bool, error) {
	out := bytes.NewBuffer(nil)
	err := c.KubectlInCluster(exec.WithWriteTo(ctx, out), "get", "pod", "--namespace=kube-system", "--output=json", c.getComponentName(name))
	if err != nil {
		if strings.Contains(out.String(), "NotFound") {
			return false, nil
		}
		return false, err
	}

	var pod corev1.Pod
	err = json.Unmarshal(out.Bytes(), &pod)
	if err != nil {
		return false, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false, nil
	}
	if pod.Status.ContainerStatuses == nil {
		return false, nil
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false, nil
		}
	}

	return true, nil
}

func (c *Cluster) getClusterName() string {
	return c.Name() + "-control-plane"
}

func (c *Cluster) getComponentName(name string) string {
	clusterName := c.getClusterName()
	switch name {
	case "prometheus":
	default:
		name = name + "-" + clusterName
	}
	return name
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	componentName := c.getComponentName(name)

	args := []string{"logs", "-n", "kube-system"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, componentName)

	err := c.Kubectl(exec.WithAllWriteTo(ctx, out), args...)
	if err != nil {
		return err
	}
	return nil
}

// Logs returns the logs of the specified component.
func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, false)
}

// LogsFollow follows the logs of the component
func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, true)
}

// ListBinaries list binaries in the cluster
func (c *Cluster) ListBinaries(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	return []string{
		conf.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	return []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
	}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	etcdContainerName := c.getComponentName("etcd")

	args = append(
		[]string{
			"exec", "-i", "-n", "kube-system", etcdContainerName, "--",
			"etcdctl",
			"--endpoints=" + net.LocalAddress + ":2379",
			"--cert=/etc/kubernetes/pki/etcd/server.crt",
			"--key=/etc/kubernetes/pki/etcd/server.key",
			"--cacert=/etc/kubernetes/pki/etcd/ca.crt",
		},
		args...,
	)
	return c.KubectlInCluster(ctx, args...)
}

// preDownloadKind pre-download and cache kind
func (c *Cluster) preDownloadKind(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	_, err = exec.LookPath("kind")
	if err != nil {
		// kind does not exist, try to download it
		kindPath := c.GetBinPath("kind" + conf.BinSuffix)
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KindBinary, kindPath, 0750, conf.QuietPull)
		if err != nil {
			return "", err
		}
		return kindPath, nil
	}

	return "kind", nil
}
