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
	"path/filepath"
	"time"

	"github.com/nxadm/tail"
	"golang.org/x/sync/errgroup"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
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
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0750, conf.QuietPull)
	if err != nil {
		return err
	}

	if !conf.DisableKubeControllerManager {
		kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if !conf.DisableKubeScheduler {
		kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KwokControllerBinary, kwokControllerPath, 0750, conf.QuietPull)
	if err != nil {
		return err
	}

	etcdPath := c.GetBinPath("etcd" + conf.BinSuffix)
	if conf.EtcdBinary == "" {
		err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd"+conf.BinSuffix, 0750, conf.QuietPull, true)
		if err != nil {
			return err
		}
	} else {
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.EtcdBinary, etcdPath, 0750, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := c.GetBinPath("prometheus" + conf.BinSuffix)
		if conf.PrometheusBinary == "" {
			err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus"+conf.BinSuffix, 0750, conf.QuietPull, true)
			if err != nil {
				return err
			}
		} else {
			err = file.DownloadWithCache(ctx, conf.CacheDir, conf.PrometheusBinary, prometheusPath, 0750, conf.QuietPull)
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
		err = pki.GeneratePki(pkiPath, sans...)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	if conf.KubeAuditPolicy != "" {
		auditLogPath := c.GetLogPath(runtime.AuditLogName)
		err = file.Create(auditLogPath, 0644)
		if err != nil {
			return err
		}

		auditPolicyPath := c.GetWorkdirPath(runtime.AuditPolicyName)
		err = file.Copy(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	err = os.MkdirAll(etcdDataPath, 0750)
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
	etcdVersion, err := version.ParseFromBinary(ctx, etcdPath)
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
	kubeApiserverVersion, err := version.ParseFromBinary(ctx, kubeApiserverPath)
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

		kubeControllerManagerVersion, err := version.ParseFromBinary(ctx, kubeControllerManagerPath)
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
			err = k8s.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, kubeconfigPath)
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

		kubeSchedulerVersion, err := version.ParseFromBinary(ctx, kubeSchedulerPath)
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
			ExtraArgs:        kubeSchedulerComponentPatches.ExtraArgs,
			ExtraVolumes:     kubeSchedulerComponentPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeSchedulerComponent)
	}

	// Configure the kwok-controller
	kwokControllerVersion, err := version.ParseFromBinary(ctx, kwokControllerPath)
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
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0640)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}

		prometheusVersion, err := version.ParseFromBinary(ctx, prometheusPath)
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
	err = os.WriteFile(kubeconfigPath, kubeconfigData, 0640)
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
	return exec.IsRunning(ctx, component.WorkDir, component.Binary)
}

func (c *Cluster) startComponent(ctx context.Context, component internalversion.Component) error {
	return exec.ForkExec(ctx, component.WorkDir, component.Binary, component.Args...)
}

func (c *Cluster) startComponents(ctx context.Context, cs []internalversion.Component) error {
	groups, err := components.GroupByLinks(cs)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	err = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		for i, group := range groups {
			if len(group) == 1 {
				if err = c.startComponent(ctx, group[0]); err != nil {
					return false, err
				}
			} else { // parallel start components
				g, ctx := errgroup.WithContext(ctx)
				for _, component := range group {
					component := component
					logger.Debug("Starting component",
						"component", component.Name,
						"group", i,
					)
					g.Go(func() error {
						return c.startComponent(ctx, component)
					})
				}
				if err := g.Wait(); err != nil {
					return false, err
				}
			}
		}

		// check apiserver is ready
		ready, err := c.Ready(ctx)
		if err != nil {
			logger.Debug("Apiserver is not ready",
				"err", err,
			)
			return false, nil
		}
		if !ready {
			logger.Debug("Apiserver is not ready")
			return false, nil
		}

		// check if all components is running
		for i, group := range groups {
			component, notReady := slices.Find(group, func(component internalversion.Component) bool {
				return !c.isRunning(ctx, component)
			})
			if notReady {
				logger.Debug("Component is not running, retrying",
					"component", component.Name,
					"group", i,
				)
				return false, nil
			}
		}
		return true, nil
	}, wait.WithTimeout(2*time.Minute), wait.WithImmediate())
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) stopComponent(ctx context.Context, component internalversion.Component) error {
	return exec.ForkExecKill(ctx, component.WorkDir, component.Binary)
}

func (c *Cluster) stopComponents(ctx context.Context, cs []internalversion.Component) error {
	groups, err := components.GroupByLinks(cs)
	if err != nil {
		return err
	}
	g, _ := errgroup.WithContext(ctx)
	for i := len(groups) - 1; i >= 0; i-- {
		group := groups[i]
		for _, component := range group {
			component := component
			g.Go(func() error {
				return c.stopComponent(ctx, component)
			})
		}
		if err := g.Wait(); err != nil {
			return err
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

	err = c.startComponents(ctx, config.Components)
	if err != nil {
		return err
	}

	return nil
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	err = c.stopComponents(ctx, config.Components)
	if err != nil {
		return err
	}

	return nil
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	err = c.startComponents(ctx, config.Components)
	if err != nil {
		return err
	}

	return nil
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	err = c.stopComponents(ctx, config.Components)
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

	logs := c.GetLogPath(filepath.Base(name) + ".log")

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

	logs := c.GetLogPath(filepath.Base(name) + ".log")

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
	etcdctlPath := c.GetBinPath("etcdctl" + conf.BinSuffix)

	err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdctlPath, "etcdctl"+conf.BinSuffix, 0750, conf.QuietPull, true)
	if err != nil {
		return err
	}

	return exec.Exec(ctx, etcdctlPath, append([]string{"--endpoints", net.LocalAddress + ":" + format.String(conf.EtcdPort)}, args...)...)
}
