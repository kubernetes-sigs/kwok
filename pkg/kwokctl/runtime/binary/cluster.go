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
	"context"
	"fmt"
	"io"
	"os"
	rt "runtime"
	"time"

	"github.com/nxadm/tail"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/wait"
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

func (c *Cluster) download(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	kubeApiserverPath := c.GetBinPath("kube-apiserver" + conf.BinSuffix)
	err = c.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0750, conf.QuietPull)
	if err != nil {
		return err
	}

	if !conf.DisableKubeControllerManager {
		kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
		err = c.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if !conf.DisableKubeScheduler {
		kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
		err = c.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)
	err = c.DownloadWithCache(ctx, conf.CacheDir, conf.KwokControllerBinary, kwokControllerPath, 0750, conf.QuietPull)
	if err != nil {
		return err
	}

	etcdPath := c.GetBinPath("etcd" + conf.BinSuffix)
	if conf.EtcdBinary == "" {
		err = c.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd"+conf.BinSuffix, 0750, conf.QuietPull, true)
		if err != nil {
			return err
		}
	} else {
		err = c.DownloadWithCache(ctx, conf.CacheDir, conf.EtcdBinary, etcdPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := c.GetBinPath("prometheus" + conf.BinSuffix)
		if conf.PrometheusBinary == "" {
			err = c.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus"+conf.BinSuffix, 0750, conf.QuietPull, true)
			if err != nil {
				return err
			}
		} else {
			err = c.DownloadWithCache(ctx, conf.CacheDir, conf.PrometheusBinary, prometheusPath, 0750, conf.QuietPull)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cluster) setup(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

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
		err = c.CreateFile(auditLogPath)
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
	err = c.MkdirAll(etcdDataPath)
	if err != nil {
		return fmt.Errorf("failed to mkdir etcd data path: %w", err)
	}

	return nil
}

func (c *Cluster) setupPorts(ctx context.Context, ports ...*uint32) error {
	for _, port := range ports {
		if port != nil && *port == 0 {
			p, err := net.GetUnusedPort(ctx)
			if err != nil {
				return err
			}
			*port = p
		}
	}
	return nil
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
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

	logger := log.FromContext(ctx)
	verbosity := logger.Level()
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	err = c.download(ctx)
	if err != nil {
		return err
	}

	err = c.setup(ctx)
	if err != nil {
		return err
	}

	scheme := "http"
	if conf.SecurePort {
		scheme = "https"
	}

	workdir := c.Workdir()

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	kubeApiserverPath := c.GetBinPath("kube-apiserver" + conf.BinSuffix)
	kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
	kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)
	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	etcdPath := c.GetBinPath("etcd" + conf.BinSuffix)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	caCertPath := path.Join(pkiPath, "ca.crt")
	adminKeyPath := path.Join(pkiPath, "admin.key")
	adminCertPath := path.Join(pkiPath, "admin.crt")
	auditLogPath := ""
	auditPolicyPath := ""

	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	err = c.setupPorts(ctx,
		&conf.EtcdPeerPort,
		&conf.EtcdPort,
		&conf.KubeApiserverPort,
		&conf.KwokControllerPort,
	)
	if err != nil {
		return err
	}

	// Configure the etcd
	etcdVersion, err := c.ParseVersionFromBinary(ctx, etcdPath)
	if err != nil {
		return err
	}

	etcdComponentPatches := runtime.GetComponentPatches(config, "etcd")
	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Workdir:      workdir,
		Binary:       etcdPath,
		Version:      etcdVersion,
		BindAddress:  conf.BindAddress,
		DataPath:     etcdDataPath,
		Port:         conf.EtcdPort,
		PeerPort:     conf.EtcdPeerPort,
		Verbosity:    verbosity,
		ExtraArgs:    etcdComponentPatches.ExtraArgs,
		ExtraVolumes: etcdComponentPatches.ExtraVolumes,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, etcdComponent)

	// Configure the kube-apiserver
	kubeApiserverVersion, err := c.ParseVersionFromBinary(ctx, kubeApiserverPath)
	if err != nil {
		return err
	}

	kubeApiserverComponentPatches := runtime.GetComponentPatches(config, "kube-apiserver")
	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Workdir:           workdir,
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
		AuditPolicyPath:   auditPolicyPath,
		AuditLogPath:      auditLogPath,
		CaCertPath:        caCertPath,
		AdminCertPath:     adminCertPath,
		AdminKeyPath:      adminKeyPath,
		Verbosity:         verbosity,
		DisableQPSLimits:  conf.DisableQPSLimits,
		ExtraArgs:         kubeApiserverComponentPatches.ExtraArgs,
		ExtraVolumes:      kubeApiserverComponentPatches.ExtraVolumes,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kubeApiserverComponent)

	// Configure the kube-controller-manager
	if !conf.DisableKubeControllerManager {
		err = c.setupPorts(ctx,
			&conf.KubeControllerManagerPort,
		)
		if err != nil {
			return err
		}

		kubeControllerManagerVersion, err := c.ParseVersionFromBinary(ctx, kubeControllerManagerPath)
		if err != nil {
			return err
		}

		kubeControllerManagerPatches := runtime.GetComponentPatches(config, "kube-controller-manager")
		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Workdir:                            workdir,
			Binary:                             kubeControllerManagerPath,
			Version:                            kubeControllerManagerVersion,
			BindAddress:                        conf.BindAddress,
			Port:                               conf.KubeControllerManagerPort,
			SecurePort:                         conf.SecurePort,
			CaCertPath:                         caCertPath,
			AdminCertPath:                      adminCertPath,
			AdminKeyPath:                       adminKeyPath,
			KubeAuthorization:                  conf.KubeAuthorization,
			KubeconfigPath:                     kubeconfigPath,
			KubeFeatureGates:                   conf.KubeFeatureGates,
			NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
			NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
			Verbosity:                          verbosity,
			DisableQPSLimits:                   conf.DisableQPSLimits,
			ExtraArgs:                          kubeControllerManagerPatches.ExtraArgs,
			ExtraVolumes:                       kubeControllerManagerPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeControllerManagerComponent)
	}

	// Configure the kube-scheduler
	if !conf.DisableKubeScheduler {
		schedulerConfigPath := ""
		if conf.KubeSchedulerConfig != "" {
			schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
			err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, kubeconfigPath)
			if err != nil {
				return err
			}
		}

		err = c.setupPorts(ctx,
			&conf.KubeSchedulerPort,
		)
		if err != nil {
			return err
		}

		kubeSchedulerVersion, err := c.ParseVersionFromBinary(ctx, kubeSchedulerPath)
		if err != nil {
			return err
		}

		kubeSchedulerComponentPatches := runtime.GetComponentPatches(config, "kube-scheduler")
		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Workdir:          workdir,
			Binary:           kubeSchedulerPath,
			Version:          kubeSchedulerVersion,
			BindAddress:      conf.BindAddress,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       caCertPath,
			AdminCertPath:    adminCertPath,
			AdminKeyPath:     adminKeyPath,
			ConfigPath:       schedulerConfigPath,
			KubeconfigPath:   kubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
			Verbosity:        verbosity,
			DisableQPSLimits: conf.DisableQPSLimits,
			ExtraArgs:        kubeSchedulerComponentPatches.ExtraArgs,
			ExtraVolumes:     kubeSchedulerComponentPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeSchedulerComponent)
	}

	// Configure the kwok-controller
	kwokControllerVersion, err := c.ParseVersionFromBinary(ctx, kwokControllerPath)
	if err != nil {
		return err
	}

	kwokControllerComponentPatches := runtime.GetComponentPatches(config, "kwok-controller")

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Workdir:                  workdir,
		Binary:                   kwokControllerPath,
		Version:                  kwokControllerVersion,
		BindAddress:              conf.BindAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               kwokConfigPath,
		KubeconfigPath:           kubeconfigPath,
		CaCertPath:               caCertPath,
		AdminCertPath:            adminCertPath,
		AdminKeyPath:             adminKeyPath,
		NodeName:                 "localhost",
		Verbosity:                verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		ExtraArgs:                kwokControllerComponentPatches.ExtraArgs,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kwokControllerComponent)

	// Configure the prometheus
	if conf.PrometheusPort != 0 {
		prometheusPath := c.GetBinPath("prometheus" + conf.BinSuffix)

		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:               c.Name(),
			SecurePort:                conf.SecurePort,
			AdminCrtPath:              adminCertPath,
			AdminKeyPath:              adminKeyPath,
			PrometheusPort:            conf.PrometheusPort,
			EtcdPort:                  conf.EtcdPort,
			KubeApiserverPort:         conf.KubeApiserverPort,
			KubeControllerManagerPort: conf.KubeControllerManagerPort,
			KubeSchedulerPort:         conf.KubeSchedulerPort,
			KwokControllerPort:        conf.KwokControllerPort,
			Metrics:                   runtime.GetMetrics(ctx),
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)
		err = c.WriteFile(prometheusConfigPath, []byte(prometheusData))
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}

		prometheusVersion, err := c.ParseVersionFromBinary(ctx, prometheusPath)
		if err != nil {
			return err
		}

		prometheusComponentPatches := runtime.GetComponentPatches(config, "prometheus")
		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Workdir:      workdir,
			Binary:       prometheusPath,
			Version:      prometheusVersion,
			BindAddress:  conf.BindAddress,
			Port:         conf.PrometheusPort,
			ConfigPath:   prometheusConfigPath,
			Verbosity:    verbosity,
			ExtraArgs:    prometheusComponentPatches.ExtraArgs,
			ExtraVolumes: prometheusComponentPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, prometheusComponent)
	}

	// Setup kubeconfig
	kubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + net.LocalAddress + ":" + format.String(conf.KubeApiserverPort),
		CACrtPath:    caCertPath,
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	}))
	if err != nil {
		return err
	}
	err = c.WriteFile(kubeconfigPath, kubeconfigData)
	if err != nil {
		return err
	}

	// Save config
	err = c.SetConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to set config", err)
	}
	err = c.Save(ctx)
	if err != nil {
		logger.Error("Failed to update cluster", err)
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
		return err
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
		conf.EtcdBinaryTar,
		conf.KubeApiserverBinary,
		conf.KubeControllerManagerBinary,
		conf.KubeSchedulerBinary,
		conf.KwokControllerBinary,
		conf.PrometheusBinaryTar,
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
