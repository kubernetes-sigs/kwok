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
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
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
	return c.Exec(ctx, c.runtime, "version")
}

// https://github.com/kubernetes-sigs/kind/blob/7b017b2ce14a7fdea9d3ed2fa259c38c927e2dd1/pkg/internal/runtime/runtime.go
func (c *Cluster) withProviderEnv(ctx context.Context) context.Context {
	provider := c.runtime
	ctx = exec.WithEnv(ctx, []string{
		"KIND_EXPERIMENTAL_PROVIDER=" + provider,
	})
	return ctx
}

type env struct {
	kwokctlConfig       *internalversion.KwokctlConfiguration
	verbosity           log.Level
	inClusterKubeconfig string
	auditLogPath        string
	auditPolicyPath     string
}

func (c *Cluster) env(ctx context.Context) (*env, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	inClusterKubeconfig := "/etc/kubernetes/scheduler.conf"
	auditLogPath := ""
	auditPolicyPath := ""
	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	return &env{
		kwokctlConfig:       config,
		verbosity:           verbosity,
		inClusterKubeconfig: inClusterKubeconfig,
		auditLogPath:        auditLogPath,
		auditPolicyPath:     auditPolicyPath,
	}, nil
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	err := c.Cluster.Install(ctx)
	if err != nil {
		return err
	}

	env, err := c.env(ctx)
	if err != nil {
		return err
	}

	err = c.addKind(ctx, env)
	if err != nil {
		return err
	}

	err = c.addDashboard(ctx, env)
	if err != nil {
		return err
	}

	err = c.addPrometheus(ctx, env)
	if err != nil {
		return err
	}

	err = c.addJaeger(ctx, env)
	if err != nil {
		return err
	}

	err = c.pullAllImages(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) addKind(ctx context.Context, env *env) (err error) {
	logger := log.FromContext(ctx)
	conf := &env.kwokctlConfig.Options
	var featureGates []string
	var runtimeConfig []string
	if conf.KubeFeatureGates != "" {
		featureGates = strings.Split(strings.ReplaceAll(conf.KubeFeatureGates, "=", ": "), ",")
	}
	if conf.KubeRuntimeConfig != "" {
		runtimeConfig = strings.Split(strings.ReplaceAll(conf.KubeRuntimeConfig, "=", ": "), ",")
	}

	if conf.KubeAuditPolicy != "" {
		err = c.MkdirAll(c.GetWorkdirPath("logs"))
		if err != nil {
			return err
		}

		err = c.CreateFile(env.auditLogPath)
		if err != nil {
			return err
		}

		err = c.CopyFile(conf.KubeAuditPolicy, env.auditPolicyPath)
		if err != nil {
			return err
		}
	}

	schedulerConfigPath := ""
	if !conf.DisableKubeScheduler && conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.inClusterKubeconfig)
		if err != nil {
			return err
		}
	}

	kubeApiserverTracingConfigPath := ""
	if conf.JaegerPort != 0 {
		kubeApiserverTracingConfigData, err := k8s.BuildKubeApiserverTracingConfig(k8s.BuildKubeApiserverTracingConfigParam{
			Endpoint: conf.BindAddress + ":4317",
		})
		if err != nil {
			return fmt.Errorf("failed to generate kubeApiserverTracingConfig yaml: %w", err)
		}
		kubeApiserverTracingConfigPath = c.GetWorkdirPath(runtime.ApiserverTracingConfig)

		err = c.WriteFile(kubeApiserverTracingConfigPath, []byte(kubeApiserverTracingConfigData))
		if err != nil {
			return fmt.Errorf("failed to write kubeApiserverTracingConfig yaml: %w", err)
		}
	}

	configPath := c.GetWorkdirPath(runtime.ConfigName)

	kubeVersion, err := version.ParseVersion(conf.KubeVersion)
	if err != nil {
		return err
	}

	etcdComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentEtcd)
	kubeApiserverComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeApiserver)
	kubeSchedulerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeScheduler)
	kubeControllerManagerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKubeControllerManager)
	kwokControllerComponentPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentKwokController)
	extraLogVolumes := runtime.GetLogVolumes(ctx)
	kwokControllerExtraVolumes := kwokControllerComponentPatches.ExtraVolumes
	kwokControllerExtraVolumes = append(kwokControllerExtraVolumes, extraLogVolumes...)
	if len(etcdComponentPatches.ExtraEnvs) > 0 ||
		len(kubeApiserverComponentPatches.ExtraEnvs) > 0 ||
		len(kubeSchedulerComponentPatches.ExtraEnvs) > 0 ||
		len(kubeControllerManagerComponentPatches.ExtraEnvs) > 0 {
		logger.Warn("extraEnvs config in etcd, kube-apiserver, kube-scheduler or kube-controller-manager is not supported in kind")
	}
	kindYaml, err := BuildKind(BuildKindConfig{
		BindAddress:                   conf.BindAddress,
		KubeApiserverPort:             conf.KubeApiserverPort,
		EtcdPort:                      conf.EtcdPort,
		JaegerPort:                    conf.JaegerPort,
		DashboardPort:                 conf.DashboardPort,
		PrometheusPort:                conf.PrometheusPort,
		KwokControllerPort:            conf.KwokControllerPort,
		FeatureGates:                  featureGates,
		RuntimeConfig:                 runtimeConfig,
		AuditPolicy:                   env.auditPolicyPath,
		AuditLog:                      env.auditLogPath,
		SchedulerConfig:               schedulerConfigPath,
		ConfigPath:                    configPath,
		TracingConfigPath:             kubeApiserverTracingConfigPath,
		Verbosity:                     env.verbosity,
		EtcdExtraArgs:                 etcdComponentPatches.ExtraArgs,
		EtcdExtraVolumes:              etcdComponentPatches.ExtraVolumes,
		ApiserverExtraArgs:            kubeApiserverComponentPatches.ExtraArgs,
		ApiserverExtraVolumes:         kubeApiserverComponentPatches.ExtraVolumes,
		SchedulerExtraArgs:            kubeSchedulerComponentPatches.ExtraArgs,
		SchedulerExtraVolumes:         kubeSchedulerComponentPatches.ExtraVolumes,
		ControllerManagerExtraArgs:    kubeControllerManagerComponentPatches.ExtraArgs,
		ControllerManagerExtraVolumes: kubeControllerManagerComponentPatches.ExtraVolumes,
		KwokControllerExtraVolumes:    kwokControllerExtraVolumes,
		KwokControllerExtraExtraEnvs:  kwokControllerComponentPatches.ExtraEnvs,
		DisableQPSLimits:              conf.DisableQPSLimits,
		KubeVersion:                   kubeVersion,
	})
	if err != nil {
		return err
	}
	err = c.WriteFile(c.GetWorkdirPath(runtime.KindName), []byte(kindYaml))
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KindName, err)
	}

	kwokControllerPod, err := BuildKwokControllerPod(BuildKwokControllerPodConfig{
		KwokControllerImage:      conf.KwokControllerImage,
		Name:                     c.Name(),
		Verbosity:                env.verbosity,
		NodeLeaseDurationSeconds: 40,
		EnableCRDs:               conf.EnableCRDs,
		ExtraArgs:                kwokControllerComponentPatches.ExtraArgs,
		ExtraVolumes:             kwokControllerExtraVolumes,
		ExtraEnvs:                kwokControllerComponentPatches.ExtraEnvs,
	})
	if err != nil {
		return err
	}
	err = c.WriteFile(c.GetWorkdirPath(runtime.KwokPod), []byte(kwokControllerPod))
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", runtime.KwokPod, err)
	}
	return nil
}

func (c *Cluster) addDashboard(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	if conf.DashboardPort != 0 {
		dashboardPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentDashboard)
		dashboardConf := BuildDashboardDeploymentConfig{
			DashboardImage: conf.DashboardImage,
			Name:           c.Name(),
			Banner:         fmt.Sprintf("Welcome to %s", c.Name()),
			ExtraArgs:      dashboardPatches.ExtraArgs,
			ExtraVolumes:   dashboardPatches.ExtraVolumes,
			ExtraEnvs:      dashboardPatches.ExtraEnvs,
		}
		dashboardDeploy, err := BuildDashboardDeployment(dashboardConf)
		if err != nil {
			return err
		}
		err = c.WriteFile(c.GetWorkdirPath(runtime.DashboardDeploy), []byte(dashboardDeploy))
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.DashboardDeploy, err)
		}
	}
	return nil
}

func (c *Cluster) addPrometheus(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	if conf.PrometheusPort != 0 {
		prometheusPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentPrometheus)
		prometheusConf := BuildPrometheusDeploymentConfig{
			PrometheusImage: conf.PrometheusImage,
			Name:            c.Name(),
			ExtraArgs:       prometheusPatches.ExtraArgs,
			ExtraVolumes:    prometheusPatches.ExtraVolumes,
			ExtraEnvs:       prometheusPatches.ExtraEnvs,
		}
		if env.verbosity != log.LevelInfo {
			prometheusConf.LogLevel = log.ToLogSeverityLevel(env.verbosity)
		}
		prometheusDeploy, err := BuildPrometheusDeployment(prometheusConf)
		if err != nil {
			return err
		}
		err = c.WriteFile(c.GetWorkdirPath(runtime.PrometheusDeploy), []byte(prometheusDeploy))
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.PrometheusDeploy, err)
		}
	}
	return nil
}

func (c *Cluster) addJaeger(_ context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	if conf.JaegerPort != 0 {
		jaegerPatches := runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentJaeger)
		jaegerConf := BuildJaegerDeploymentConfig{
			JaegerImage:  conf.JaegerImage,
			Name:         c.Name(),
			ExtraArgs:    jaegerPatches.ExtraArgs,
			ExtraVolumes: jaegerPatches.ExtraVolumes,
			ExtraEnvs:    jaegerPatches.ExtraEnvs,
		}
		if env.verbosity != log.LevelInfo {
			jaegerConf.LogLevel = log.ToLogSeverityLevel(env.verbosity)
		}
		jaegerDeploy, err := BuildJaegerDeployment(jaegerConf)
		if err != nil {
			return err
		}
		err = c.WriteFile(c.GetWorkdirPath(runtime.JaegerDeploy), []byte(jaegerDeploy))
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", runtime.JaegerDeploy, err)
		}
	}
	return nil
}

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	config.Components = append(config.Components,
		internalversion.Component{
			Name: consts.ComponentEtcd,
		},
		internalversion.Component{
			Name: consts.ComponentKubeApiserver,
		},
		internalversion.Component{
			Name: consts.ComponentKwokController,
		},
	)

	if conf.DashboardPort != 0 {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: consts.ComponentDashboard,
			},
		)
	}

	if conf.PrometheusPort != 0 {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: consts.ComponentPrometheus,
			},
		)
	}

	if conf.JaegerPort != 0 {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: consts.ComponentJaeger,
			},
		)
	}

	if !conf.DisableKubeScheduler {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: consts.ComponentKubeScheduler,
			},
		)
	}

	if !conf.DisableKubeControllerManager {
		config.Components = append(config.Components,
			internalversion.Component{
				Name: consts.ComponentKubeControllerManager,
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

	err = c.Exec(exec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath, args...)
	if err != nil {
		return err
	}

	// TODO: remove this when kind support set server
	err = c.fillKubeconfigContextServer(conf.BindAddress)
	if err != nil {
		return err
	}

	err = c.loadAllImages(ctx)
	if err != nil {
		return err
	}

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)

	kubeconfigBuf := bytes.NewBuffer(nil)
	err = c.Kubectl(exec.WithWriteTo(ctx, kubeconfigBuf), "config", "view", "--minify=true", "--raw=true")
	if err != nil {
		return err
	}

	err = c.WriteFile(kubeconfigPath, kubeconfigBuf.Bytes())
	if err != nil {
		return err
	}

	kindName := c.getClusterName()
	err = c.Exec(ctx, c.runtime, "cp", c.GetWorkdirPath(runtime.KwokPod), kindName+":/etc/kubernetes/manifests/kwok-controller.yaml")
	if err != nil {
		return err
	}

	// Copy ca.crt and ca.key to host's workdir
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	err = c.MkdirAll(pkiPath)
	if err != nil {
		return err
	}
	err = c.Exec(ctx, c.runtime, "cp", kindName+":/etc/kubernetes/pki/ca.crt", path.Join(pkiPath, "ca.crt"))
	if err != nil {
		return err
	}
	err = c.Exec(ctx, c.runtime, "cp", kindName+":/etc/kubernetes/pki/ca.key", path.Join(pkiPath, "ca.key"))
	if err != nil {
		return err
	}

	if conf.DashboardPort != 0 {
		err = c.Kubectl(exec.WithAllWriteToErrOut(ctx), "apply", "-f", c.GetWorkdirPath(runtime.DashboardDeploy))
		if err != nil {
			return err
		}
	}

	if conf.PrometheusPort != 0 {
		err = c.Kubectl(exec.WithAllWriteToErrOut(ctx), "apply", "-f", c.GetWorkdirPath(runtime.PrometheusDeploy))
		if err != nil {
			return err
		}
	}
	if conf.JaegerPort != 0 {
		err = c.Kubectl(exec.WithAllWriteToErrOut(ctx), "apply", "-f", c.GetWorkdirPath(runtime.JaegerDeploy))
		if err != nil {
			return err
		}
	}

	// Cordoning the node to prevent fake pods from being scheduled on it
	err = c.Kubectl(ctx, "cordon", c.getClusterName())
	if err != nil {
		logger.Error("Failed cordon node", err)
	}

	if conf.DisableKubeScheduler {
		err := c.StopComponent(ctx, consts.ComponentKubeScheduler)
		if err != nil {
			logger.Error("Failed to disable kube-scheduler", err)
		}
	}

	if conf.DisableKubeControllerManager {
		err := c.StopComponent(ctx, consts.ComponentKubeControllerManager)
		if err != nil {
			logger.Error("Failed to disable kube-controller-manager", err)
		}
	}

	return nil
}

func (c *Cluster) pullAllImages(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options
	images := []string{
		conf.KindNodeImage,
		conf.KwokControllerImage,
	}
	if conf.DashboardPort != 0 {
		images = append(images, conf.DashboardImage)
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	if conf.JaegerPort != 0 {
		images = append(images, conf.JaegerImage)
	}
	err := c.PullImages(ctx, c.runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) loadAllImages(ctx context.Context) error {
	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options
	images := []string{conf.KwokControllerImage}
	if conf.DashboardPort != 0 {
		images = append(images, conf.DashboardImage)
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	if conf.JaegerPort != 0 {
		images = append(images, conf.JaegerImage)
	}

	if c.runtime == consts.RuntimeTypeDocker {
		err = c.loadDockerImages(ctx, kindPath, c.Name(), images)
	} else {
		err = c.loadArchiveImages(ctx, kindPath, c.Name(), images, c.runtime, conf.CacheDir)
	}
	if err != nil {
		return err
	}
	return nil
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
		logger.Info("Loaded image", "image", image)
	}
	return nil
}

// loadArchiveImages loads docker images into the cluster.
// `kind load image-archive`
func (c *Cluster) loadArchiveImages(ctx context.Context, command string, kindCluster string, images []string, runtime string, tmpDir string) error {
	logger := log.FromContext(ctx)
	for _, image := range images {
		archive := path.Join(tmpDir, "image-archive", strings.ReplaceAll(image, ":", "/")+".tar")
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
		logger.Info("Loaded image", "image", image)
		err = c.Remove(archive)
		if err != nil {
			return err
		}
	}
	return nil
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
	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	err = c.Exec(exec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath, "delete", "cluster", "--name", c.Name())
	if err != nil {
		logger.Error("Failed to delete cluster", err)
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

var importantComponents = map[string]struct{}{
	consts.ComponentEtcd:          {},
	consts.ComponentKubeApiserver: {},
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", name)
	if _, important := importantComponents[name]; !important {
		if !c.IsDryRun() {
			if _, exist, err := c.inspectComponent(ctx, name); err != nil {
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
	if c.IsDryRun() {
		return nil
	}
	return c.waitComponentReady(ctx, name, true, 120*time.Second)
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", name)
	if _, important := importantComponents[name]; !important {
		if !c.IsDryRun() {
			if _, exist, err := c.inspectComponent(ctx, name); err != nil {
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
	if _, important := importantComponents[name]; important {
		return nil
	}
	if c.IsDryRun() {
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
		exist   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, exist, err = c.inspectComponent(ctx, name)
		if err != nil {
			logger.Debug("check component ready",
				"component", name,
				"err", err,
			)
			//nolint:nilerr
			return false, nil
		}
		if wantReady {
			return ready, nil
		}
		return !exist, nil
	},
		wait.WithTimeout(timeout),
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

func (c *Cluster) inspectComponent(ctx context.Context, name string) (ready bool, exist bool, err error) {
	out := bytes.NewBuffer(nil)
	err = c.KubectlInCluster(exec.WithWriteTo(ctx, out), "get", "pod", "--namespace=kube-system", "--output=json", c.getComponentName(name))
	if err != nil {
		if strings.Contains(out.String(), "NotFound") {
			return false, false, nil
		}
		return false, false, err
	}

	var pod corev1.Pod
	err = json.Unmarshal(out.Bytes(), &pod)
	if err != nil {
		return false, true, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false, true, nil
	}
	if pod.Status.ContainerStatuses == nil {
		return false, true, nil
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false, true, nil
		}
	}

	return true, true, nil
}

func (c *Cluster) getClusterName() string {
	return c.Name() + "-control-plane"
}

func (c *Cluster) getComponentName(name string) string {
	clusterName := c.getClusterName()
	switch name {
	case consts.ComponentPrometheus:
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
	if c.IsDryRun() && !follow {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessage("%s >%s", runtime.FormatExec(ctx, name, args...), file)
			return nil
		}
	}

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

// CollectLogs returns the logs of the specified component.
func (c *Cluster) CollectLogs(ctx context.Context, dir string) error {
	logger := log.FromContext(ctx)

	kwokConfigPath := path.Join(dir, "kwok.yaml")
	if file.Exists(kwokConfigPath) {
		return fmt.Errorf("%s already exists", kwokConfigPath)
	}

	if err := c.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}
	logger.Info("Exporting logs", "dir", dir)

	err := c.CopyFile(c.GetWorkdirPath(runtime.ConfigName), kwokConfigPath)
	if err != nil {
		return err
	}

	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	componentsDir := path.Join(dir, "components")
	err = c.MkdirAll(componentsDir)
	if err != nil {
		return err
	}

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	infoPath := path.Join(dir, conf.Options.Runtime+"-info.txt")
	err = c.WriteToPath(c.withProviderEnv(ctx), infoPath, []string{kindPath, "version"})
	if err != nil {
		return err
	}

	for _, component := range conf.Components {
		logPath := path.Join(componentsDir, component.Name+".log")
		f, err := c.OpenFile(logPath)
		if err != nil {
			logger.Error("Failed to open file", err)
			continue
		}
		if err = c.Logs(ctx, component.Name, f); err != nil {
			logger.Error("Failed to get log", err)
			if err = f.Close(); err != nil {
				logger.Error("Failed to close file", err)
				if err = c.Remove(logPath); err != nil {
					logger.Error("Failed to remove file", err)
				}
			}
		}
		if err = f.Close(); err != nil {
			logger.Error("Failed to close file", err)
			if err = c.Remove(logPath); err != nil {
				logger.Error("Failed to remove file", err)
			}
		}
	}

	if conf.Options.KubeAuditPolicy != "" {
		filePath := path.Join(componentsDir, "audit.log")
		f, err := c.OpenFile(filePath)
		if err != nil {
			logger.Error("Failed to open file", err)
		} else {
			if err = c.AuditLogs(ctx, f); err != nil {
				logger.Error("Failed to get audit log", err)
			}
			if err = f.Close(); err != nil {
				logger.Error("Failed to close file", err)
				if err = c.Remove(filePath); err != nil {
					logger.Error("Failed to remove file", err)
				}
			}
		}
	}

	return nil
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
	etcdContainerName := c.getComponentName(consts.ComponentEtcd)

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
		err = c.DownloadWithCache(ctx, conf.CacheDir, conf.KindBinary, kindPath, 0750, conf.QuietPull)
		if err != nil {
			return "", err
		}
		return kindPath, nil
	}

	return "kind", nil
}
