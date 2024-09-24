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
	"fmt"
	"io"
	"os"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// Cluster is an implementation of Runtime for docker.
type Cluster struct {
	*runtime.Cluster

	runtime string

	isNerdctl               bool
	canNerdctlUnlessStopped *bool
}

// NewDockerCluster creates a new Runtime for docker.
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
}

// NewPodmanCluster creates a new Runtime for podman.
func NewPodmanCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypePodman,
	}, nil
}

// NewNerdctlCluster creates a new Runtime for nerdctl.
func NewNerdctlCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeNerdctl,
		isNerdctl: true,
	}, nil
}

// NewLimaCluster creates a new Runtime for lima.
func NewLimaCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeNerdctl + "." + consts.RuntimeTypeLima,
		isNerdctl: true,
	}, nil
}

// NewFinchCluster creates a new Runtime for finch.
func NewFinchCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeFinch,
		isNerdctl: true,
	}, nil
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	if c.IsDryRun() {
		return nil
	}
	return c.Exec(ctx, c.runtime, "version")
}

func (c *Cluster) setup(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options
	if !file.Exists(env.pkiPath) {
		sans := []string{
			c.Name() + "-kube-apiserver",
			c.Name() + "-kwok-controller",
		}
		ips, err := net.GetAllIPs()
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Warn("failed to get all ips", "err", err)
		} else {
			sans = append(sans, ips...)
		}
		if len(conf.KubeApiserverCertSANs) != 0 {
			sans = append(sans, conf.KubeApiserverCertSANs...)
		}
		err = c.MkdirAll(env.pkiPath)
		if err != nil {
			return fmt.Errorf("failed to create pki dir: %w", err)
		}
		err = c.GeneratePki(env.pkiPath, sans...)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	if conf.KubeAuditPolicy != "" {
		err := c.MkdirAll(c.GetWorkdirPath("logs"))
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

	err := c.MkdirAll(env.etcdDataPath)
	if err != nil {
		return fmt.Errorf("failed to mkdir etcd data path: %w", err)
	}

	return nil
}

func (c *Cluster) setupPorts(ctx context.Context, used sets.Sets[uint32], ports ...*uint32) error {
	for _, port := range ports {
		if port != nil && *port == 0 {
			p, err := net.GetUnusedPort(ctx, used)
			if err != nil {
				return err
			}
			*port = p
		}
	}
	return nil
}

type env struct {
	kwokctlConfig                 *internalversion.KwokctlConfiguration
	components                    []string
	verbosity                     log.Level
	inClusterOnHostKubeconfigPath string
	inClusterKubeconfig           string
	kubeconfigPath                string
	etcdDataPath                  string
	kwokConfigPath                string
	pkiPath                       string
	auditLogPath                  string
	auditPolicyPath               string
	workdir                       string
	caCertPath                    string
	adminKeyPath                  string
	adminCertPath                 string
	inClusterPkiPath              string
	inClusterCaCertPath           string
	inClusterAdminKeyPath         string
	inClusterAdminCertPath        string
	inClusterPort                 uint32
	scheme                        string
	usedPorts                     sets.Sets[uint32]
}

func (c *Cluster) env(ctx context.Context) (*env, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	components, err := c.Components(ctx)
	if err != nil {
		return nil, err
	}

	inClusterOnHostKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
	inClusterKubeconfig := "/root/.kube/config"
	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	auditLogPath := ""
	auditPolicyPath := ""
	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	workdir := c.Workdir()
	caCertPath := path.Join(pkiPath, "ca.crt")
	adminKeyPath := path.Join(pkiPath, "admin.key")
	adminCertPath := path.Join(pkiPath, "admin.crt")
	inClusterPkiPath := "/etc/kubernetes/pki/"
	inClusterCaCertPath := path.Join(inClusterPkiPath, "ca.crt")
	inClusterAdminKeyPath := path.Join(inClusterPkiPath, "admin.key")
	inClusterAdminCertPath := path.Join(inClusterPkiPath, "admin.crt")

	inClusterPort := uint32(8080)
	scheme := "http"
	if config.Options.SecurePort {
		scheme = "https"
		inClusterPort = 6443
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	usedPorts := runtime.GetUsedPorts(ctx)

	return &env{
		kwokctlConfig:                 config,
		components:                    components,
		verbosity:                     verbosity,
		inClusterOnHostKubeconfigPath: inClusterOnHostKubeconfigPath,
		inClusterKubeconfig:           inClusterKubeconfig,
		kubeconfigPath:                kubeconfigPath,
		etcdDataPath:                  etcdDataPath,
		kwokConfigPath:                kwokConfigPath,
		pkiPath:                       pkiPath,
		auditLogPath:                  auditLogPath,
		auditPolicyPath:               auditPolicyPath,
		workdir:                       workdir,
		caCertPath:                    caCertPath,
		adminKeyPath:                  adminKeyPath,
		adminCertPath:                 adminCertPath,
		inClusterPkiPath:              inClusterPkiPath,
		inClusterCaCertPath:           inClusterCaCertPath,
		inClusterAdminKeyPath:         inClusterAdminKeyPath,
		inClusterAdminCertPath:        inClusterAdminCertPath,
		inClusterPort:                 inClusterPort,
		scheme:                        scheme,
		usedPorts:                     usedPorts,
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

	err = c.preInstall(ctx, env)
	if err != nil {
		return err
	}

	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&env.kwokctlConfig.Options.KubeApiserverPort,
	)
	if err != nil {
		return err
	}

	err = c.addEtcd(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeApiserver(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubectlProxy(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeControllerManager(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeScheduler(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKwokController(ctx, env)
	if err != nil {
		return err
	}

	err = c.addMetricsServer(ctx, env)
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

	err = c.addDashboard(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPrometheusConfig(ctx, env)
	if err != nil {
		return err
	}

	err = c.finishInstall(ctx, env)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) addEtcd(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the etcd
	err = c.EnsureImage(ctx, c.runtime, conf.EtcdImage)
	if err != nil {
		return err
	}
	etcdVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.EtcdImage, "etcd")
	if err != nil {
		return err
	}

	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Image:            conf.EtcdImage,
		Version:          etcdVersion,
		BindAddress:      net.PublicAddress,
		Port:             conf.EtcdPort,
		DataPath:         env.etcdDataPath,
		Verbosity:        env.verbosity,
		QuotaBackendSize: conf.EtcdQuotaBackendSize,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, etcdComponent)
	return nil
}

func (c *Cluster) addKubeApiserver(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kube-apiserver
	err = c.EnsureImage(ctx, c.runtime, conf.KubeApiserverImage)
	if err != nil {
		return err
	}
	kubeApiserverVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeApiserverImage, consts.ComponentKubeApiserver)
	if err != nil {
		return err
	}

	kubeApiserverTracingConfigPath := ""
	if conf.JaegerPort != 0 {
		kubeApiserverTracingConfigData, err := k8s.BuildKubeApiserverTracingConfig(k8s.BuildKubeApiserverTracingConfigParam{
			Endpoint: c.Name() + "-jaeger:4317",
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

	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Runtime:           conf.Runtime,
		ProjectName:       c.Name(),
		Workdir:           env.workdir,
		Image:             conf.KubeApiserverImage,
		Version:           kubeApiserverVersion,
		BindAddress:       net.PublicAddress,
		Port:              conf.KubeApiserverPort,
		KubeRuntimeConfig: conf.KubeRuntimeConfig,
		KubeFeatureGates:  conf.KubeFeatureGates,
		SecurePort:        conf.SecurePort,
		KubeAuthorization: conf.KubeAuthorization,
		KubeAdmission:     conf.KubeAdmission,
		AuditPolicyPath:   env.auditPolicyPath,
		AuditLogPath:      env.auditLogPath,
		CaCertPath:        env.caCertPath,
		AdminCertPath:     env.adminCertPath,
		AdminKeyPath:      env.adminKeyPath,
		EtcdPort:          conf.EtcdPort,
		EtcdAddress:       c.Name() + "-etcd",
		Verbosity:         env.verbosity,
		DisableQPSLimits:  conf.DisableQPSLimits,
		TracingConfigPath: kubeApiserverTracingConfigPath,
		EtcdPrefix:        conf.EtcdPrefix,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeApiserverComponent)
	return nil
}

func (c *Cluster) addKubectlProxy(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeApiserverInsecureProxy) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kubectl
	err = c.EnsureImage(ctx, c.runtime, conf.KubectlImage)
	if err != nil {
		return err
	}

	kubectlProxyComponent, err := components.BuildKubectlProxyComponent(components.BuildKubectlProxyComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.KubectlImage,
		BindAddress:    net.PublicAddress,
		Port:           conf.KubeApiserverInsecurePort,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubectlProxyComponent)
	return nil
}

func (c *Cluster) addKubeControllerManager(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeControllerManager) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kube-controller-manager
	err = c.EnsureImage(ctx, c.runtime, conf.KubeControllerManagerImage)
	if err != nil {
		return err
	}
	kubeControllerManagerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeControllerManagerImage, consts.ComponentKubeControllerManager)
	if err != nil {
		return err
	}

	kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
		Runtime:                            conf.Runtime,
		ProjectName:                        c.Name(),
		Workdir:                            env.workdir,
		Image:                              conf.KubeControllerManagerImage,
		Version:                            kubeControllerManagerVersion,
		BindAddress:                        net.PublicAddress,
		Port:                               conf.KubeControllerManagerPort,
		SecurePort:                         conf.SecurePort,
		CaCertPath:                         env.caCertPath,
		AdminCertPath:                      env.adminCertPath,
		AdminKeyPath:                       env.adminKeyPath,
		KubeAuthorization:                  conf.KubeAuthorization,
		KubeconfigPath:                     env.inClusterOnHostKubeconfigPath,
		KubeFeatureGates:                   conf.KubeFeatureGates,
		Verbosity:                          env.verbosity,
		DisableQPSLimits:                   conf.DisableQPSLimits,
		NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
		NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeControllerManagerComponent)
	return nil
}

func (c *Cluster) addKubeScheduler(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeScheduler) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kube-scheduler
	schedulerConfigPath := ""
	if conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.inClusterKubeconfig)
		if err != nil {
			return err
		}
	}

	err = c.EnsureImage(ctx, c.runtime, conf.KubeSchedulerImage)
	if err != nil {
		return err
	}
	kubeSchedulerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeSchedulerImage, consts.ComponentKubeScheduler)
	if err != nil {
		return err
	}

	kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Image:            conf.KubeSchedulerImage,
		Version:          kubeSchedulerVersion,
		BindAddress:      net.PublicAddress,
		Port:             conf.KubeSchedulerPort,
		SecurePort:       conf.SecurePort,
		CaCertPath:       env.caCertPath,
		AdminCertPath:    env.adminCertPath,
		AdminKeyPath:     env.adminKeyPath,
		ConfigPath:       schedulerConfigPath,
		KubeconfigPath:   env.inClusterOnHostKubeconfigPath,
		KubeFeatureGates: conf.KubeFeatureGates,
		Verbosity:        env.verbosity,
		DisableQPSLimits: conf.DisableQPSLimits,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeSchedulerComponent)
	return nil
}

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKwokController) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kwok-controller
	err = c.EnsureImage(ctx, c.runtime, conf.KwokControllerImage)
	if err != nil {
		return err
	}

	kwokControllerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	logVolumes := runtime.GetLogVolumes(ctx)

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                  conf.Runtime,
		ProjectName:              c.Name(),
		Workdir:                  env.workdir,
		Image:                    conf.KwokControllerImage,
		Version:                  kwokControllerVersion,
		BindAddress:              net.PublicAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               env.kwokConfigPath,
		KubeconfigPath:           env.inClusterOnHostKubeconfigPath,
		CaCertPath:               env.caCertPath,
		AdminCertPath:            env.adminCertPath,
		AdminKeyPath:             env.adminKeyPath,
		NodeName:                 c.Name() + "-kwok-controller",
		Verbosity:                env.verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		EnableCRDs:               conf.EnableCRDs,
	})
	kwokControllerComponent.Volumes = append(kwokControllerComponent.Volumes, logVolumes...)

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}

func (c *Cluster) addMetricsServer(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentMetricsServer) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	err = c.EnsureImage(ctx, c.runtime, conf.MetricsServerImage)
	if err != nil {
		return err
	}

	metricsServerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.MetricsServerImage, consts.ComponentMetricsServer)
	if err != nil {
		return err
	}

	metricsServerComponent, err := components.BuildMetricsServerComponent(components.BuildMetricsServerComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.MetricsServerImage,
		Version:        metricsServerVersion,
		BindAddress:    conf.BindAddress,
		Port:           conf.MetricsServerPort,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, metricsServerComponent)
	return nil
}

func (c *Cluster) setupPrometheusConfig(_ context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentPrometheus) {
		return nil
	}

	// Configure the prometheus
	prometheusData, err := components.BuildPrometheus(components.BuildPrometheusConfig{
		Components: env.kwokctlConfig.Components,
	})
	if err != nil {
		return fmt.Errorf("failed to generate prometheus yaml: %w", err)
	}
	prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

	// We don't need to check the permissions of the prometheus config file,
	// because it's working in a non-root container.
	err = c.WriteFileWithMode(prometheusConfigPath, []byte(prometheusData), 0644)
	if err != nil {
		return fmt.Errorf("failed to write prometheus yaml: %w", err)
	}
	return nil
}

func (c *Cluster) addPrometheus(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentPrometheus) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the prometheus
	err = c.EnsureImage(ctx, c.runtime, conf.PrometheusImage)
	if err != nil {
		return err
	}

	prometheusVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.PrometheusImage, "")
	if err != nil {
		return err
	}

	prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

	prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
		Runtime:                      conf.Runtime,
		Workdir:                      env.workdir,
		Image:                        conf.PrometheusImage,
		Version:                      prometheusVersion,
		BindAddress:                  net.PublicAddress,
		Port:                         conf.PrometheusPort,
		ConfigPath:                   prometheusConfigPath,
		AdminCertPath:                env.adminCertPath,
		AdminKeyPath:                 env.adminKeyPath,
		Verbosity:                    env.verbosity,
		DisableKubeControllerManager: !slices.Contains(env.components, consts.ComponentKubeControllerManager),
		DisableKubeScheduler:         !slices.Contains(env.components, consts.ComponentKubeScheduler),
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, prometheusComponent)
	return nil
}

func (c *Cluster) addDashboard(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentDashboard) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	err = c.EnsureImage(ctx, c.runtime, conf.DashboardImage)
	if err != nil {
		return err
	}
	dashboardVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.DashboardImage, "")
	if err != nil {
		return err
	}

	enableMetricsServer := slices.Contains(env.components, consts.ComponentMetricsServer)
	dashboardComponent, err := components.BuildDashboardComponent(components.BuildDashboardComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.DashboardImage,
		Version:        dashboardVersion,
		BindAddress:    net.PublicAddress,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		Port:           conf.DashboardPort,
		Banner:         fmt.Sprintf("Welcome to %s", c.Name()),
		EnableMetrics:  enableMetricsServer,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardComponent)

	if enableMetricsServer {
		err = c.EnsureImage(ctx, c.runtime, conf.DashboardMetricsScraperImage)
		if err != nil {
			return err
		}
		dashboardMetricsScraperComponent, err := components.BuildDashboardMetricsScraperComponent(components.BuildDashboardMetricsScraperComponentConfig{
			Runtime:        conf.Runtime,
			Workdir:        env.workdir,
			Image:          conf.DashboardMetricsScraperImage,
			KubeconfigPath: env.inClusterOnHostKubeconfigPath,
			CaCertPath:     env.caCertPath,
			AdminCertPath:  env.adminCertPath,
			AdminKeyPath:   env.adminKeyPath,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardMetricsScraperComponent)
	}
	return nil
}

func (c *Cluster) addJaeger(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentJaeger) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the jaeger
	err = c.EnsureImage(ctx, c.runtime, conf.JaegerImage)
	if err != nil {
		return err
	}

	jaegerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.JaegerImage, "")
	if err != nil {
		return err
	}

	jaegerComponent, err := components.BuildJaegerComponent(components.BuildJaegerComponentConfig{
		Runtime:      conf.Runtime,
		Workdir:      env.workdir,
		Image:        conf.JaegerImage,
		Version:      jaegerVersion,
		BindAddress:  net.PublicAddress,
		Port:         conf.JaegerPort,
		OtlpGrpcPort: conf.JaegerOtlpGrpcPort,
		Verbosity:    env.verbosity,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, jaegerComponent)
	return nil
}

func (c *Cluster) preInstall(_ context.Context, env *env) error {
	for i, patch := range env.kwokctlConfig.ComponentsPatches {
		if len(patch.ExtraVolumes) == 0 {
			continue
		}
		volumes, err := runtime.ExpandVolumesHostPaths(patch.ExtraVolumes)
		if err != nil {
			return fmt.Errorf("failed to expand host volumes for %q component: %w", patch.Name, err)
		}

		env.kwokctlConfig.ComponentsPatches[i].ExtraVolumes = volumes
	}
	return nil
}

func (c *Cluster) finishInstall(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	for i := range env.kwokctlConfig.Components {
		runtime.ApplyComponentPatches(ctx, &env.kwokctlConfig.Components[i], env.kwokctlConfig.ComponentsPatches)
	}

	// Setup kubeconfig
	kubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      env.scheme + "://" + net.LocalAddress + ":" + format.String(conf.KubeApiserverPort),
		CACrtPath:    env.caCertPath,
		AdminCrtPath: env.adminCertPath,
		AdminKeyPath: env.adminKeyPath,
	}))
	if err != nil {
		return err
	}

	inClusterKubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      env.scheme + "://" + c.Name() + "-kube-apiserver:" + format.String(env.inClusterPort),
		CACrtPath:    env.inClusterCaCertPath,
		AdminCrtPath: env.inClusterAdminCertPath,
		AdminKeyPath: env.inClusterAdminKeyPath,
	}))
	if err != nil {
		return err
	}

	// Save config
	err = c.WriteFile(env.kubeconfigPath, kubeconfigData)
	if err != nil {
		return err
	}

	err = c.WriteFile(env.inClusterOnHostKubeconfigPath, inClusterKubeconfigData)
	if err != nil {
		return err
	}

	err = c.SetConfig(ctx, env.kwokctlConfig)
	if err != nil {
		return err
	}
	err = c.Save(ctx)
	if err != nil {
		return err
	}

	err = c.createNetwork(ctx)
	if err != nil {
		return err
	}

	err = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err = c.createComponents(ctx)
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

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := c.deleteComponents(ctx)
		return err == nil, err
	},
		wait.WithContinueOnError(5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}

	err = c.deleteNetwork(ctx)
	if err != nil {
		return err
	}

	err = c.Cluster.Uninstall(ctx)
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
	if c.isNerdctl {
		canNerdctlUnlessStopped, _ := c.isCanNerdctlUnlessStopped(ctx)
		if !canNerdctlUnlessStopped {
			// TODO: Remove this, nerdctl stop will restart containers
			// https://github.com/containerd/nerdctl/issues/1980
			err := c.createComponents(ctx)
			if err != nil {
				return err
			}
		}
	}
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

	if c.isNerdctl {
		canNerdctlUnlessStopped, _ := c.isCanNerdctlUnlessStopped(ctx)
		if !canNerdctlUnlessStopped {
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
	}
	return nil
}

func (c *Cluster) stop(ctx context.Context) error {
	if c.isNerdctl {
		canNerdctlUnlessStopped, _ := c.isCanNerdctlUnlessStopped(ctx)
		if !canNerdctlUnlessStopped {
			err := c.SnapshotSave(ctx, c.GetWorkdirPath("restart.db"))
			if err != nil {
				return fmt.Errorf("failed to snapshot cluster data: %w", err)
			}
		}
	}
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
	if c.isNerdctl {
		canNerdctlUnlessStopped, _ := c.isCanNerdctlUnlessStopped(ctx)
		if !canNerdctlUnlessStopped {
			// TODO: Remove this, nerdctl stop will restart containers
			// https://github.com/containerd/nerdctl/issues/1980
			err = c.deleteComponents(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, componentName string) error {
	return c.startComponent(ctx, componentName)
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, componentName string) error {
	return c.stopComponent(ctx, componentName)
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, c.Name()+"-"+name)
	if c.IsDryRun() && !follow {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessage("%s >%s", runtime.FormatExec(ctx, name, args...), file)
			return nil
		}
	}

	err := c.Exec(exec.WithAllWriteTo(ctx, out), c.runtime, args...)
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

	infoPath := path.Join(dir, c.runtime+"-info.txt")
	err = c.WriteToPath(ctx, infoPath, []string{c.runtime, "info"})
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
		conf.EtcdImage,
		conf.KubeApiserverImage,
		conf.KubeControllerManagerImage,
		conf.KubeSchedulerImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
		conf.MetricsServerImage,
	}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	etcdContainerName := c.Name() + "-etcd"

	// If using versions earlier than v3.4, set `ETCDCTL_API=3` to use v3 API.
	args = append([]string{"exec", "--env=ETCDCTL_API=3", "-i", etcdContainerName, "etcdctl"}, args...)
	return c.Exec(ctx, c.runtime, args...)
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

// InitCRs initializes the CRs.
func (c *Cluster) InitCRs(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	_, enableMetricsServer := slices.Find(config.Components, func(c internalversion.Component) bool {
		return c.Name == consts.ComponentMetricsServer
	})
	if c.IsDryRun() {
		if enableMetricsServer {
			dryrun.PrintMessage("# Set up apiservice for metrics server")
		}

		return nil
	}

	buf := bytes.NewBuffer(nil)
	if enableMetricsServer {
		apiservice, err := components.BuildMetricsServerAPIService(components.BuildMetricsServerAPIServiceConfig{
			Port:         4443,
			ExternalName: c.Name() + "-metrics-server",
		})
		if err != nil {
			return err
		}
		_, _ = buf.WriteString(apiservice)
		_, _ = buf.WriteString("---\n")
	}

	if buf.Len() == 0 {
		return nil
	}

	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return err
	}

	loader, err := snapshot.NewLoader(snapshot.LoadConfig{
		Clientset: clientset,
		NoFilers:  true,
	})
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(buf)

	return loader.Load(ctx, decoder)
}
