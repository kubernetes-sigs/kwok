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
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
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

func (c *Cluster) setup(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	if !file.Exists(pkiPath) {
		sans := []string{}
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
		err = c.MkdirAll(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to create pki dir: %w", err)
		}
		err = c.GeneratePki(pkiPath, sans...)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	pkiEtcd := filepath.Join(pkiPath, "etcd")
	err := c.MkdirAll(pkiEtcd)
	if err != nil {
		return fmt.Errorf("failed to create pki dir: %w", err)
	}
	return nil
}

// https://github.com/kubernetes-sigs/kind/blob/7b017b2ce14a7fdea9d3ed2fa259c38c927e2dd1/pkg/internal/runtime/runtime.go
func (c *Cluster) withProviderEnv(ctx context.Context) context.Context {
	provider := c.runtime
	ctx = exec.WithEnv(ctx, []string{
		"KIND_EXPERIMENTAL_PROVIDER=" + provider,
	})
	return ctx
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
	kwokctlConfig        *internalversion.KwokctlConfiguration
	verbosity            log.Level
	schedulerConfigPath  string
	auditLogPath         string
	auditPolicyPath      string
	prometheusConfigPath string

	inClusterOnHostKubeconfigPath string
	workdir                       string
	caCertPath                    string
	adminKeyPath                  string
	adminCertPath                 string

	kwokConfigPath string

	usedPorts sets.Sets[uint32]
}

func (c *Cluster) env(ctx context.Context) (*env, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	inClusterOnHostKubeconfigPath := "/etc/kubernetes/admin.conf"
	schedulerConfigPath := "/etc/kubernetes/scheduler.conf"
	prometheusConfigPath := "/etc/prometheus/prometheus.yaml"
	kwokConfigPath := "/etc/kwok/kwok.yaml"
	auditLogPath := ""
	auditPolicyPath := ""
	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	pkiPath := "/etc/kubernetes/pki"

	workdir := c.Workdir()
	caCertPath := path.Join(pkiPath, "ca.crt")
	adminKeyPath := path.Join(pkiPath, "admin.key")
	adminCertPath := path.Join(pkiPath, "admin.crt")

	usedPorts := runtime.GetUsedPorts(ctx)
	return &env{
		kwokctlConfig:                 config,
		verbosity:                     verbosity,
		schedulerConfigPath:           schedulerConfigPath,
		prometheusConfigPath:          prometheusConfigPath,
		auditLogPath:                  auditLogPath,
		auditPolicyPath:               auditPolicyPath,
		inClusterOnHostKubeconfigPath: inClusterOnHostKubeconfigPath,
		workdir:                       workdir,
		caCertPath:                    caCertPath,
		adminKeyPath:                  adminKeyPath,
		adminCertPath:                 adminCertPath,
		kwokConfigPath:                kwokConfigPath,
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

	// This is not necessary when creating a cluster use kind, but in Linux the cluster is created as root,
	// and the files here may not have permissions when deleted, so we create them first.
	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&env.kwokctlConfig.Options.KubeApiserverPort,
		&env.kwokctlConfig.Options.EtcdPort,
	)
	if err != nil {
		return err
	}

	err = c.addKind(ctx, env)
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

	err = c.addDashboard(ctx, env)
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

	err = c.setupPrometheusConfig(ctx, env)
	if err != nil {
		return err
	}

	images, err := c.listAllImages(ctx)
	if err != nil {
		return err
	}

	err = c.pullAllImages(ctx, env, images)
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

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	err = c.MkdirAll(pkiPath)
	if err != nil {
		return err
	}

	manifestsPath := c.GetWorkdirPath(runtime.ManifestsName)
	err = c.MkdirAll(manifestsPath)
	if err != nil {
		return err
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
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.schedulerConfigPath)
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

	var prometheusPatches internalversion.ComponentPatches
	if conf.PrometheusPort != 0 {
		prometheusPatches = runtime.GetComponentPatches(env.kwokctlConfig, consts.ComponentPrometheus)
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

		prometheusPatches.ExtraVolumes = append(prometheusPatches.ExtraVolumes, internalversion.Volume{
			Name:      "prometheus-config",
			HostPath:  prometheusConfigPath,
			MountPath: env.prometheusConfigPath,
		})
	}

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
		TracingConfigPath:             kubeApiserverTracingConfigPath,
		Workdir:                       c.Workdir(),
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
		PrometheusExtraVolumes:        prometheusPatches.ExtraVolumes,
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

	return nil
}

func (c *Cluster) addEtcd(_ context.Context, env *env) (err error) {
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, internalversion.Component{
		Name: consts.ComponentEtcd,
		Metric: &internalversion.ComponentMetric{
			Scheme:             "https",
			Host:               "127.0.0.1:2379",
			Path:               "/metrics",
			CertPath:           "/etc/kubernetes/pki/apiserver-etcd-client.crt",
			KeyPath:            "/etc/kubernetes/pki/apiserver-etcd-client.key",
			InsecureSkipVerify: true,
		},
	})
	return nil
}

func (c *Cluster) addKubeApiserver(_ context.Context, env *env) (err error) {
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, internalversion.Component{
		Name: consts.ComponentKubeApiserver,
		Metric: &internalversion.ComponentMetric{
			Scheme:             "https",
			Host:               "127.0.0.1:6443",
			Path:               "/metrics",
			CertPath:           "/etc/kubernetes/pki/admin.crt",
			KeyPath:            "/etc/kubernetes/pki/admin.key",
			InsecureSkipVerify: true,
		},
	})
	return nil
}

func (c *Cluster) addKubeControllerManager(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options
	if !conf.DisableKubeControllerManager {
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, internalversion.Component{
			Name: consts.ComponentKubeControllerManager,
			Metric: &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               "127.0.0.1:10257",
				Path:               "/metrics",
				CertPath:           "/etc/kubernetes/pki/admin.crt",
				KeyPath:            "/etc/kubernetes/pki/admin.key",
				InsecureSkipVerify: true,
			},
		})
	}
	return nil
}

func (c *Cluster) addKubeScheduler(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options
	if !conf.DisableKubeScheduler {
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, internalversion.Component{
			Name: consts.ComponentKubeScheduler,
			Metric: &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               "127.0.0.1:10259",
				Path:               "/metrics",
				CertPath:           "/etc/kubernetes/pki/admin.crt",
				KeyPath:            "/etc/kubernetes/pki/admin.key",
				InsecureSkipVerify: true,
			},
		})
	}
	return nil
}

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kwok-controller
	kwokControllerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	logVolumes := runtime.GetLogVolumes(ctx)
	logVolumes = slices.Map(logVolumes, func(v internalversion.Volume) internalversion.Volume {
		v.HostPath = path.Join("/var/components/controller", v.HostPath)
		return v
	})

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                           conf.Runtime,
		ProjectName:                       c.Name(),
		Workdir:                           env.workdir,
		Image:                             conf.KwokControllerImage,
		Version:                           kwokControllerVersion,
		BindAddress:                       net.PublicAddress,
		Port:                              conf.KwokControllerPort,
		ConfigPath:                        env.kwokConfigPath,
		KubeconfigPath:                    env.inClusterOnHostKubeconfigPath,
		CaCertPath:                        env.caCertPath,
		AdminCertPath:                     env.adminCertPath,
		AdminKeyPath:                      env.adminKeyPath,
		NodeIP:                            "$(POD_IP)",
		NodeName:                          "kwok-controller.kube-system.svc",
		ManageNodesWithAnnotationSelector: "kwok.x-k8s.io/node=fake",
		Verbosity:                         env.verbosity,
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        conf.EnableCRDs,
	})
	kwokControllerComponent.Volumes = append(kwokControllerComponent.Volumes, logVolumes...)

	runtime.ApplyComponentPatches(&kwokControllerComponent, env.kwokctlConfig.ComponentsPatches)

	pod := components.ConvertToPod(kwokControllerComponent)
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
	kwokControllerPod, err := yaml.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal kwok controller pod: %w", err)
	}
	err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentKwokController+".yaml"), kwokControllerPod)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}

func (c *Cluster) addDashboard(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	if conf.DashboardPort != 0 {
		dashboardVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.DashboardImage, "")
		if err != nil {
			return err
		}

		dashboardComponent, err := components.BuildDashboardComponent(components.BuildDashboardComponentConfig{
			Runtime:        conf.Runtime,
			Workdir:        env.workdir,
			Image:          conf.DashboardImage,
			Version:        dashboardVersion,
			BindAddress:    net.PublicAddress,
			KubeconfigPath: env.inClusterOnHostKubeconfigPath,
			CaCertPath:     env.caCertPath,
			AdminCertPath:  env.adminCertPath,
			AdminKeyPath:   env.adminKeyPath,
			Port:           8080,
			Banner:         fmt.Sprintf("Welcome to %s", c.Name()),
			EnableMetrics:  conf.EnableMetricsServer,
		})
		if err != nil {
			return fmt.Errorf("failed to build dashboard component: %w", err)
		}

		runtime.ApplyComponentPatches(&dashboardComponent, env.kwokctlConfig.ComponentsPatches)

		dashboardPod, err := yaml.Marshal(components.ConvertToPod(dashboardComponent))
		if err != nil {
			return fmt.Errorf("failed to marshal dashboard pod: %w", err)
		}
		err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentDashboard+".yaml"), dashboardPod)
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardComponent)

		if conf.EnableMetricsServer {
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
			dashboardMetricsScraperPod, err := yaml.Marshal(components.ConvertToPod(dashboardMetricsScraperComponent))
			if err != nil {
				return fmt.Errorf("failed to marshal dashboard metrics scraper pod: %w", err)
			}
			err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentDashboardMetricsScraper+".yaml"), dashboardMetricsScraperPod)
			if err != nil {
				return fmt.Errorf("failed to write: %w", err)
			}
			env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardMetricsScraperComponent)
		}
	}
	return nil
}

func (c *Cluster) addMetricsServer(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options
	if conf.EnableMetricsServer {
		metricsServerVersion, err := c.ParseVersionFromImage(context.Background(), c.runtime, conf.MetricsServerImage, "")
		if err != nil {
			return err
		}

		metricsServerComponent, err := components.BuildMetricsServerComponent(components.BuildMetricsServerComponentConfig{
			Runtime:        conf.Runtime,
			ProjectName:    c.Name(),
			Workdir:        env.workdir,
			Image:          conf.MetricsServerImage,
			Version:        metricsServerVersion,
			BindAddress:    net.PublicAddress,
			Port:           443,
			CaCertPath:     env.caCertPath,
			AdminCertPath:  env.adminCertPath,
			AdminKeyPath:   env.adminKeyPath,
			KubeconfigPath: env.inClusterOnHostKubeconfigPath,
			Verbosity:      env.verbosity,
		})
		if err != nil {
			return err
		}

		metricsServerPod, err := yaml.Marshal(components.ConvertToPod(metricsServerComponent))
		if err != nil {
			return fmt.Errorf("failed to marshal metrics server pod: %w", err)
		}
		err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentMetricsServer+".yaml"), metricsServerPod)
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, metricsServerComponent)
	}
	return nil
}

func (c *Cluster) setupPrometheusConfig(_ context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the prometheus
	if conf.PrometheusPort != 0 {
		prometheusData, err := components.BuildPrometheus(components.BuildPrometheusConfig{
			Components: env.kwokctlConfig.Components,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)
		err = c.WriteFile(prometheusConfigPath, []byte(prometheusData))
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}
	}
	return nil
}

func (c *Cluster) addPrometheus(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	if conf.PrometheusPort != 0 {
		prometheusVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.PrometheusImage, "")
		if err != nil {
			return err
		}

		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Runtime:       conf.Runtime,
			Workdir:       env.workdir,
			Image:         conf.PrometheusImage,
			Version:       prometheusVersion,
			BindAddress:   net.PublicAddress,
			Port:          9090,
			ConfigPath:    "/var/components/prometheus/etc/prometheus/prometheus.yaml",
			AdminCertPath: env.adminCertPath,
			AdminKeyPath:  env.adminKeyPath,
			Verbosity:     env.verbosity,
		})
		if err != nil {
			return err
		}

		prometheusComponent.Volumes = append(prometheusComponent.Volumes,
			internalversion.Volume{
				HostPath:  "/etc/kubernetes/pki/apiserver-etcd-client.crt",
				MountPath: "/etc/kubernetes/pki/apiserver-etcd-client.crt",
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  "/etc/kubernetes/pki/apiserver-etcd-client.key",
				MountPath: "/etc/kubernetes/pki/apiserver-etcd-client.key",
				ReadOnly:  true,
			},
		)

		runtime.ApplyComponentPatches(&prometheusComponent, env.kwokctlConfig.ComponentsPatches)

		prometheusPod, err := yaml.Marshal(components.ConvertToPod(prometheusComponent))
		if err != nil {
			return fmt.Errorf("failed to marshal prometheus pod: %w", err)
		}
		err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentPrometheus+".yaml"), prometheusPod)
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, prometheusComponent)
	}
	return nil
}

func (c *Cluster) addJaeger(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	if conf.JaegerPort != 0 {
		jaegerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.JaegerImage, "")
		if err != nil {
			return err
		}

		jaegerComponent, err := components.BuildJaegerComponent(components.BuildJaegerComponentConfig{
			Runtime:     conf.Runtime,
			Workdir:     env.workdir,
			Image:       conf.JaegerImage,
			Version:     jaegerVersion,
			BindAddress: net.PublicAddress,
			Port:        16686,
			Verbosity:   env.verbosity,
		})
		if err != nil {
			return err
		}

		runtime.ApplyComponentPatches(&jaegerComponent, env.kwokctlConfig.ComponentsPatches)

		jaegerPod, err := yaml.Marshal(components.ConvertToPod(jaegerComponent))
		if err != nil {
			return fmt.Errorf("failed to marshal jaeger pod: %w", err)
		}
		err = c.WriteFile(path.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentJaeger+".yaml"), jaegerPod)
		if err != nil {
			return fmt.Errorf("failed to write: %w", err)
		}

		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, jaegerComponent)
	}
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

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	logger := log.FromContext(ctx)

	if conf.DisableKubeScheduler {
		defer func() {
			err := c.StopComponent(ctx, consts.ComponentKubeScheduler)
			if err != nil {
				logger.Error("Failed to disable kube-scheduler", err)
			}
		}()
	}

	if conf.DisableKubeControllerManager {
		defer func() {
			err := c.StopComponent(ctx, consts.ComponentKubeScheduler)
			if err != nil {
				logger.Error("Failed to disable kube-scheduler", err)
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

	err = c.Exec(exec.WithAllWriteToErrOut(c.withProviderEnv(ctx)), kindPath, args...)
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
	err = c.Kubectl(exec.WithWriteTo(ctx, kubeconfigBuf), "config", "view", "--minify=true", "--raw=true")
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
		logger.Error("Failed cordon node", err)
	}

	err = c.Exec(ctx, c.runtime, "exec", c.getClusterName(), "chmod", "-R", "+r", "/etc/kubernetes/pki")
	if err != nil {
		logger.Error("Failed to chmod pki", err)
	}

	return nil
}

func (c *Cluster) pullAllImages(ctx context.Context, env *env, images []string) error {
	conf := &env.kwokctlConfig.Options
	images = append([]string{
		conf.KindNodeImage,
	}, images...)

	err := c.PullImages(ctx, c.runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}
	return nil
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

	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	for _, component := range config.Components {
		ready, _, err := c.inspectComponent(ctx, component.Name)
		if err != nil {
			return false, err
		}

		if !ready {
			return false, nil
		}
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
	logger = logger.With("component", name)
	if _, important := startImportantComponents[name]; !important {
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
	logger = logger.With("component", name)
	if _, important := stopImportantComponents[name]; !important {
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
	if _, important := stopImportantComponents[name]; important {
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
	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return false, false, err
	}

	restConfig, err := clientset.ToRESTConfig()
	if err != nil {
		return false, false, err
	}

	typedClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, false, err
	}

	pod, err := typedClient.CoreV1().
		Pods(metav1.NamespaceSystem).
		Get(ctx, c.getComponentName(name), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, false, nil
		}
		return false, false, err
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
	return name + "-" + clusterName
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
		conf.MetricsServerImage,
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

// InitCRs initializes the CRs.
func (c *Cluster) InitCRs(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := config.Options

	if c.IsDryRun() {
		if conf.EnableMetricsServer {
			dryrun.PrintMessage("# Set up apiservice for metrics server")
		}

		return nil
	}

	buf := bytes.NewBuffer(nil)
	if conf.EnableMetricsServer {
		apiservice, err := components.BuildMetricsServerAPIService(components.BuildMetricsServerAPIServiceConfig{
			Port:         4443,
			ExternalName: "localhost",
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

	return snapshot.Load(ctx, clientset, bytes.NewBuffer(buf.Bytes()), snapshot.LoadConfig{})
}
