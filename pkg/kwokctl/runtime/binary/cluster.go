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

package binary

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/user"
	rt "runtime"
	"strconv"
	"time"

	"github.com/nxadm/tail"

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

// Cluster is an implementation of Runtime for binary
type Cluster struct {
	*runtime.Cluster
}

// NewCluster creates a new Runtime for binary
func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	return nil
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

	if conf.KubeAuditPolicy != "" {
		auditLogPath := c.GetLogPath(runtime.AuditLogName)
		err := c.CreateFile(auditLogPath)
		if err != nil {
			return err
		}

		auditPolicyPath := c.GetWorkdirPath(runtime.AuditPolicyName)
		err = c.CopyFile(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	err := c.MkdirAll(etcdDataPath)
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
	kwokctlConfig   *internalversion.KwokctlConfiguration
	verbosity       log.Level
	kubeconfigPath  string
	etcdDataPath    string
	kwokConfigPath  string
	pkiPath         string
	auditLogPath    string
	auditPolicyPath string
	workdir         string
	caCertPath      string
	adminKeyPath    string
	adminCertPath   string
	scheme          string
	usedPorts       sets.Sets[uint32]
}

func (c *Cluster) env(ctx context.Context) (*env, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	scheme := "http"
	if config.Options.SecurePort {
		scheme = "https"
	}

	workdir := c.Workdir()

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	caCertPath := path.Join(pkiPath, "ca.crt")
	adminKeyPath := path.Join(pkiPath, "admin.key")
	adminCertPath := path.Join(pkiPath, "admin.crt")
	auditLogPath := ""
	auditPolicyPath := ""

	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	usedPorts := runtime.GetUsedPorts(ctx)

	return &env{
		kwokctlConfig:   config,
		verbosity:       verbosity,
		kubeconfigPath:  kubeconfigPath,
		etcdDataPath:    etcdDataPath,
		kwokConfigPath:  kwokConfigPath,
		pkiPath:         pkiPath,
		auditLogPath:    auditLogPath,
		auditPolicyPath: auditPolicyPath,
		workdir:         workdir,
		caCertPath:      caCertPath,
		adminKeyPath:    adminKeyPath,
		adminCertPath:   adminCertPath,
		scheme:          scheme,
		usedPorts:       usedPorts,
	}, nil
}

func (c *Cluster) checkRunInCluster(ctx context.Context) {
	if !file.Exists("/var/run/secrets/kubernetes.io/serviceaccount/token") {
		return
	}

	logger := log.FromContext(ctx)
	logger.Warn("cluster may not work correctly and need to be workaround." +
		"see https://kwok.sigs.k8s.io/docs/user/all-in-one-image/#use-in-a-pod")
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	c.checkRunInCluster(ctx)

	err := c.Cluster.Install(ctx)
	if err != nil {
		return err
	}

	dirs := []string{
		"pids",
		"logs",
	}

	for _, dir := range dirs {
		err = c.MkdirAll(c.GetWorkdirPath(dir))
		if err != nil {
			return err
		}
	}

	env, err := c.env(ctx)
	if err != nil {
		return err
	}

	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&env.kwokctlConfig.Options.EtcdPeerPort,
		&env.kwokctlConfig.Options.EtcdPort,
		&env.kwokctlConfig.Options.KubeApiserverPort,
		&env.kwokctlConfig.Options.KwokControllerPort,
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
	etcdPath, err := c.EnsureBinary(ctx, consts.ComponentEtcd, conf.EtcdBinary)
	if err != nil {
		return err
	}

	etcdVersion, err := c.ParseVersionFromBinary(ctx, etcdPath)
	if err != nil {
		return err
	}

	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Runtime:     conf.Runtime,
		ProjectName: c.Name(),
		Workdir:     env.workdir,
		Binary:      etcdPath,
		Version:     etcdVersion,
		BindAddress: conf.BindAddress,
		DataPath:    env.etcdDataPath,
		Port:        conf.EtcdPort,
		PeerPort:    conf.EtcdPeerPort,
		Verbosity:   env.verbosity,
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
	kubeApiserverPath, err := c.EnsureBinary(ctx, consts.ComponentKubeApiserver, conf.KubeApiserverBinary)
	if err != nil {
		return err
	}

	kubeApiserverVersion, err := c.ParseVersionFromBinary(ctx, kubeApiserverPath)
	if err != nil {
		return err
	}

	kubeApiserverTracingConfigPath := ""
	if conf.JaegerPort != 0 {
		err = c.setupPorts(ctx,
			env.usedPorts,
			&conf.JaegerOtlpGrpcPort,
		)
		if err != nil {
			return err
		}

		kubeApiserverTracingConfigData, err := k8s.BuildKubeApiserverTracingConfig(k8s.BuildKubeApiserverTracingConfigParam{
			Endpoint: net.LocalAddress + ":" + format.String(conf.JaegerOtlpGrpcPort),
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
		Binary:            kubeApiserverPath,
		Version:           kubeApiserverVersion,
		BindAddress:       conf.BindAddress,
		Port:              conf.KubeApiserverPort,
		EtcdAddress:       net.LocalAddress,
		EtcdPort:          conf.EtcdPort,
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

func (c *Cluster) addKubeControllerManager(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kube-controller-manager
	if !conf.DisableKubeControllerManager {
		kubeControllerManagerPath, err := c.EnsureBinary(ctx, consts.ComponentKubeControllerManager, conf.KubeControllerManagerBinary)
		if err != nil {
			return err
		}

		err = c.setupPorts(ctx,
			env.usedPorts,
			&conf.KubeControllerManagerPort,
		)
		if err != nil {
			return err
		}

		kubeControllerManagerVersion, err := c.ParseVersionFromBinary(ctx, kubeControllerManagerPath)
		if err != nil {
			return err
		}

		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Runtime:                            conf.Runtime,
			ProjectName:                        c.Name(),
			Workdir:                            env.workdir,
			Binary:                             kubeControllerManagerPath,
			Version:                            kubeControllerManagerVersion,
			BindAddress:                        conf.BindAddress,
			Port:                               conf.KubeControllerManagerPort,
			SecurePort:                         conf.SecurePort,
			CaCertPath:                         env.caCertPath,
			AdminCertPath:                      env.adminCertPath,
			AdminKeyPath:                       env.adminKeyPath,
			KubeAuthorization:                  conf.KubeAuthorization,
			KubeconfigPath:                     env.kubeconfigPath,
			KubeFeatureGates:                   conf.KubeFeatureGates,
			NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
			NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
			Verbosity:                          env.verbosity,
			DisableQPSLimits:                   conf.DisableQPSLimits,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeControllerManagerComponent)
	}
	return nil
}

func (c *Cluster) addKubeScheduler(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kube-scheduler
	if !conf.DisableKubeScheduler {
		kubeSchedulerPath, err := c.EnsureBinary(ctx, consts.ComponentKubeScheduler, conf.KubeSchedulerBinary)
		if err != nil {
			return err
		}

		schedulerConfigPath := ""
		if conf.KubeSchedulerConfig != "" {
			schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
			err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.kubeconfigPath)
			if err != nil {
				return err
			}
		}

		err = c.setupPorts(ctx,
			env.usedPorts,
			&conf.KubeSchedulerPort,
		)
		if err != nil {
			return err
		}

		kubeSchedulerVersion, err := c.ParseVersionFromBinary(ctx, kubeSchedulerPath)
		if err != nil {
			return err
		}

		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Runtime:          conf.Runtime,
			ProjectName:      c.Name(),
			Workdir:          env.workdir,
			Binary:           kubeSchedulerPath,
			Version:          kubeSchedulerVersion,
			BindAddress:      conf.BindAddress,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       env.caCertPath,
			AdminCertPath:    env.adminCertPath,
			AdminKeyPath:     env.adminKeyPath,
			ConfigPath:       schedulerConfigPath,
			KubeconfigPath:   env.kubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
			Verbosity:        env.verbosity,
			DisableQPSLimits: conf.DisableQPSLimits,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeSchedulerComponent)
	}
	return nil
}

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kwok-controller
	kwokControllerPath, err := c.EnsureBinary(ctx, consts.ComponentKwokController, conf.KwokControllerBinary)
	if err != nil {
		return err
	}

	kwokControllerVersion, err := c.ParseVersionFromBinary(ctx, kwokControllerPath)
	if err != nil {
		return err
	}

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                  conf.Runtime,
		ProjectName:              c.Name(),
		Workdir:                  env.workdir,
		Binary:                   kwokControllerPath,
		Version:                  kwokControllerVersion,
		BindAddress:              conf.BindAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               env.kwokConfigPath,
		KubeconfigPath:           env.kubeconfigPath,
		CaCertPath:               env.caCertPath,
		AdminCertPath:            env.adminCertPath,
		AdminKeyPath:             env.adminKeyPath,
		NodeName:                 "localhost",
		Verbosity:                env.verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		EnableCRDs:               conf.EnableCRDs,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}

func (c *Cluster) addMetricsServer(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	if conf.EnableMetricsServer {
		metricsServerPath, err := c.EnsureBinary(ctx, consts.ComponentMetricsServer, conf.MetricsServerBinary)
		if err != nil {
			return err
		}

		metricsServerVersion, err := c.ParseVersionFromBinary(ctx, metricsServerPath)
		if err != nil {
			return err
		}

		err = c.setupPorts(ctx,
			env.usedPorts,
			&conf.MetricsServerPort,
		)
		if err != nil {
			return err
		}

		metricsServerComponent, err := components.BuildMetricsServerComponent(components.BuildMetricsServerComponentConfig{
			Runtime:        conf.Runtime,
			ProjectName:    c.Name(),
			Workdir:        env.workdir,
			Binary:         metricsServerPath,
			Version:        metricsServerVersion,
			BindAddress:    conf.BindAddress,
			Port:           conf.MetricsServerPort,
			CaCertPath:     env.caCertPath,
			AdminCertPath:  env.adminCertPath,
			AdminKeyPath:   env.adminKeyPath,
			KubeconfigPath: env.kubeconfigPath,
			Verbosity:      env.verbosity,
		})
		if err != nil {
			return err
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

	// Configure the prometheus
	if conf.PrometheusPort != 0 {
		prometheusPath, err := c.EnsureBinary(ctx, consts.ComponentPrometheus, conf.PrometheusBinary)
		if err != nil {
			return err
		}

		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

		prometheusVersion, err := c.ParseVersionFromBinary(ctx, prometheusPath)
		if err != nil {
			return err
		}

		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Runtime:     conf.Runtime,
			Workdir:     env.workdir,
			Binary:      prometheusPath,
			Version:     prometheusVersion,
			BindAddress: conf.BindAddress,
			Port:        conf.PrometheusPort,
			ConfigPath:  prometheusConfigPath,
			Verbosity:   env.verbosity,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, prometheusComponent)
	}
	return nil
}

func (c *Cluster) addJaeger(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	// Configure the jaeger
	if conf.JaegerPort != 0 {
		jaegerPath, err := c.EnsureBinary(ctx, consts.ComponentJaeger, conf.JaegerBinary)
		if err != nil {
			return err
		}

		jaegerVersion, err := c.ParseVersionFromBinary(ctx, jaegerPath)
		if err != nil {
			return err
		}

		jaegerComponent, err := components.BuildJaegerComponent(components.BuildJaegerComponentConfig{
			Runtime:      conf.Runtime,
			Workdir:      env.workdir,
			Binary:       jaegerPath,
			Version:      jaegerVersion,
			BindAddress:  conf.BindAddress,
			Port:         conf.JaegerPort,
			OtlpGrpcPort: conf.JaegerOtlpGrpcPort,
			Verbosity:    env.verbosity,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, jaegerComponent)
	}
	return nil
}

func (c *Cluster) finishInstall(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	for i := range env.kwokctlConfig.Components {
		runtime.ApplyComponentPatches(&env.kwokctlConfig.Components[i], env.kwokctlConfig.ComponentsPatches)
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
	err = c.WriteFile(env.kubeconfigPath, kubeconfigData)
	if err != nil {
		return err
	}

	// Save config
	err = c.SetConfig(ctx, env.kwokctlConfig)
	if err != nil {
		return err
	}
	err = c.Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	err := c.Cluster.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) isRunning(ctx context.Context, component internalversion.Component) bool {
	return c.ForkExecIsRunning(ctx, component.WorkDir, component.Binary)
}

func (c *Cluster) startComponent(ctx context.Context, component internalversion.Component) error {
	logger := log.FromContext(ctx)
	logger = logger.With("component", component.Name)
	if c.isRunning(ctx, component) {
		logger.Debug("Component already started")
		return nil
	}

	if len(component.Envs) > 0 {
		ctx = exec.WithEnv(ctx, slices.Map(component.Envs, func(c internalversion.Env) string {
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
		ctx = exec.WithUser(ctx, &uid, &gid)
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
	logger = logger.With("component", component.Name)
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
			logger.Warn("Cluster is not served yet", "err", err)
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

// Logs returns the logs of the specified component.
func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	_, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	logs := c.GetLogPath(name + ".log")
	if c.IsDryRun() {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessage("cp %s %s", logs, file)
		} else {
			dryrun.PrintMessage("cat %s", logs)
		}
		return nil
	}

	f, err := os.OpenFile(logs, os.O_RDONLY, 0640)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", logs, err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
		}
	}()

	_, err = io.Copy(out, f)
	if err != nil {
		return err
	}
	return nil
}

// LogsFollow follows the logs of the component
func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	_, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	logs := c.GetLogPath(name + ".log")
	if c.IsDryRun() {
		dryrun.PrintMessage("tail -f %s", logs)
		return nil
	}

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	defer func() {
		err = t.Stop()
		if err != nil {
			logger.Error("Failed to stop tail file", err)
		}
	}()

	go func() {
		for line := range t.Lines {
			_, err = out.Write([]byte(line.Text + "\n"))
			if err != nil {
				logger.Error("Failed to write line text", err)
			}
		}
	}()
	<-ctx.Done()
	return nil
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

	infoPath := path.Join(dir, consts.RuntimeTypeBinary+"-info.txt")
	f, err := c.OpenFile(infoPath)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(fmt.Sprintf("%s/%s", rt.GOOS, rt.GOARCH)))
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	for _, component := range conf.Components {
		src := c.GetLogPath(component.Name + ".log")
		dest := path.Join(componentsDir, component.Name+".log")
		if err = c.CopyFile(src, dest); err != nil {
			logger.Error("Failed to copy file", err)
		}
	}
	if conf.Options.KubeAuditPolicy != "" {
		src := c.GetLogPath(runtime.AuditLogName)
		dest := path.Join(componentsDir, runtime.AuditLogName)
		if err = c.CopyFile(src, dest); err != nil {
			logger.Error("Failed to copy file", err)
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
		conf.EtcdBinary,
		conf.KubeApiserverBinary,
		conf.KubeControllerManagerBinary,
		conf.KubeSchedulerBinary,
		conf.KwokControllerBinary,
		conf.PrometheusBinary,
		conf.MetricsServerBinary,
		conf.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options
	return c.Etcdctl(ctx, append([]string{"--endpoints", net.LocalAddress + ":" + format.String(conf.EtcdPort)}, args...)...)
}

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	for _, component := range config.Components {
		if !c.isRunning(ctx, component) {
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
			Port:         conf.MetricsServerPort,
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
