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
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

func (c *Cluster) download(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	kubeApiserverPath := c.GetBinPath("kube-apiserver" + conf.BinSuffix)
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)
	err = file.DownloadWithCache(ctx, conf.CacheDir, conf.KwokControllerBinary, kwokControllerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	etcdPath := c.GetBinPath("etcd" + conf.BinSuffix)
	if conf.EtcdBinary == "" {
		err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd"+conf.BinSuffix, 0755, conf.QuietPull, true)
		if err != nil {
			return err
		}
	} else {
		err = file.DownloadWithCache(ctx, conf.CacheDir, conf.EtcdBinary, etcdPath, 0755, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := c.GetBinPath("prometheus" + conf.BinSuffix)
		if conf.PrometheusBinary == "" {
			err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus"+conf.BinSuffix, 0755, conf.QuietPull, true)
			if err != nil {
				return err
			}
		} else {
			err = file.DownloadWithCache(ctx, conf.CacheDir, conf.PrometheusBinary, prometheusPath, 0755, conf.QuietPull)
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
		err = pki.GeneratePki(pkiPath)
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

func (c *Cluster) Install(ctx context.Context) error {
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
	localAddress := "127.0.0.1"
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
	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Workdir:  workdir,
		Binary:   etcdPath,
		Version:  etcdVersion,
		Address:  localAddress,
		DataPath: etcdDataPath,
		Port:     conf.EtcdPort,
		PeerPort: conf.EtcdPeerPort,
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
	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Workdir:           workdir,
		Binary:            kubeApiserverPath,
		Version:           kubeApiserverVersion,
		Port:              conf.KubeApiserverPort,
		EtcdAddress:       localAddress,
		EtcdPort:          conf.EtcdPort,
		KubeRuntimeConfig: conf.KubeRuntimeConfig,
		KubeFeatureGates:  conf.KubeFeatureGates,
		SecurePort:        conf.SecurePort,
		KubeAuthorization: conf.KubeAuthorization,
		AuditPolicyPath:   auditPolicyPath,
		AuditLogPath:      auditLogPath,
		CaCertPath:        caCertPath,
		AdminCertPath:     adminCertPath,
		AdminKeyPath:      adminKeyPath,
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
		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Workdir:           workdir,
			Binary:            kubeControllerManagerPath,
			Version:           kubeControllerManagerVersion,
			Address:           localAddress,
			Port:              conf.KubeControllerManagerPort,
			SecurePort:        conf.SecurePort,
			CaCertPath:        caCertPath,
			AdminKeyPath:      adminKeyPath,
			KubeAuthorization: conf.KubeAuthorization,
			KubeconfigPath:    kubeconfigPath,
			KubeFeatureGates:  conf.KubeFeatureGates,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeControllerManagerComponent)
	}

	// Configure the kube-scheduler
	if !conf.DisableKubeScheduler {
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
		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Workdir:          workdir,
			Binary:           kubeSchedulerPath,
			Version:          kubeSchedulerVersion,
			Address:          localAddress,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       caCertPath,
			AdminKeyPath:     adminKeyPath,
			KubeconfigPath:   kubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
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
	kwokControllerComponent, err := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Workdir:        workdir,
		Binary:         kwokControllerPath,
		Version:        kwokControllerVersion,
		Port:           conf.KwokControllerPort,
		ConfigPath:     kwokConfigPath,
		KubeconfigPath: kubeconfigPath,
		AdminCertPath:  adminCertPath,
		AdminKeyPath:   adminKeyPath,
		NodeName:       "localhost",
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
		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Workdir:    workdir,
			Binary:     prometheusPath,
			Version:    prometheusVersion,
			Address:    localAddress,
			Port:       conf.PrometheusPort,
			ConfigPath: prometheusConfigPath,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, prometheusComponent)
	}

	// Setup kubeconfig
	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + localAddress + ":" + format.String(conf.KubeApiserverPort),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0640)
	if err != nil {
		return err
	}

	// Save config
	logger := log.FromContext(ctx)
	err = c.SetConfig(ctx, config)
	if err != nil {
		logger.Error("Failed to set config", err)
	}
	err = c.Save(ctx)
	if err != nil {
		logger.Error("Failed to update cluster", err)
	}

	// set the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".server", scheme+"://"+localAddress+":"+format.String(conf.KubeApiserverPort))
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".cluster", c.Name())
	if conf.SecurePort {
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".insecure-skip-tls-verify", "true")
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".user", c.Name())
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-certificate", adminCertPath)
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-key", adminKeyPath)
	}
	return nil
}

func (c *Cluster) Uninstall(ctx context.Context) error {
	// unset the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "clusters."+c.Name())
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "users."+c.Name())
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "contexts."+c.Name())

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

	err = wait.PollImmediateUntilWithContext(ctx, 1*time.Second, func(ctx context.Context) (bool, error) {
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
	})
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
func (c *Cluster) EtcdctlInCluster(ctx context.Context, stm exec.IOStreams, args ...string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options
	etcdctlPath := c.GetBinPath("etcdctl" + conf.BinSuffix)

	err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdctlPath, "etcdctl"+conf.BinSuffix, 0755, conf.QuietPull, true)
	if err != nil {
		return err
	}

	return exec.Exec(ctx, "", stm, etcdctlPath, append([]string{"--endpoints", "127.0.0.1:" + format.String(conf.EtcdPort)}, args...)...)
}
