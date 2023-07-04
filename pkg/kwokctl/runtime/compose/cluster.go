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
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/envs"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// Cluster is an implementation of Runtime for docker.
type Cluster struct {
	*runtime.Cluster

	runtime string

	selfCompose *bool

	composeCommands []string

	canNerdctlUnlessStopped *bool
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
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeNerdctl,
	}, nil
}

// NewDockerCluster creates a new Runtime for docker.
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
}

var (
	selfComposePrefer = envs.GetEnvWithPrefix("CONTAINER_SELF_COMPOSE", "auto")
)

// getSwitchStatus parses the value to bool pointer.
func getSwitchStatus(value string) (*bool, error) {
	if strings.ToLower(value) == "auto" {
		return nil, nil
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (c *Cluster) isSelfCompose(ctx context.Context, creating bool) bool {
	if c.selfCompose != nil {
		return *c.selfCompose
	}

	var err error
	logger := log.FromContext(ctx)

	c.selfCompose, err = getSwitchStatus(selfComposePrefer)
	if err != nil {
		logger.Warn("Failed to parse env KWOK_CONTAINER_SELF_COMPOSE, ignore it, fallback to auto", "err", err)
	} else if c.selfCompose != nil {
		logger.Info("Found env KWOK_CONTAINER_SELF_COMPOSE, use it", "value", *c.selfCompose)
		return *c.selfCompose
	}

	if creating {
		// When create a new cluster, then use self-compose.
		c.selfCompose = format.Ptr(true)
	} else {
		// otherwise, check whether the compose file exists.
		// If not exists, then use self-compose.
		// If exists, then use *-compose.
		composePath := c.GetWorkdirPath(runtime.ComposeName)
		c.selfCompose = format.Ptr(!file.Exists(composePath))
	}

	return *c.selfCompose
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	return c.Exec(ctx, c.runtime, "version")
}

func (c *Cluster) setup(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	if !file.Exists(pkiPath) {
		sans := []string{
			c.Name() + "-kube-apiserver",
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
		err = c.MkdirAll(c.GetWorkdirPath("logs"))
		if err != nil {
			return err
		}

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

	logger := log.FromContext(ctx)
	verbosity := logger.Level()
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	err = c.setup(ctx)
	if err != nil {
		return err
	}

	inClusterOnHostKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
	inClusterKubeconfig := "/root/.kube/config"
	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
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
	if conf.SecurePort {
		scheme = "https"
		inClusterPort = 6443
	}

	err = c.setupPorts(ctx,
		&conf.KubeApiserverPort,
	)
	if err != nil {
		return err
	}

	images := []string{
		conf.EtcdImage,
		conf.KubeApiserverImage,
		conf.KwokControllerImage,
	}
	if !conf.DisableKubeControllerManager {
		images = append(images, conf.KubeControllerManagerImage)
	}
	if !conf.DisableKubeScheduler {
		images = append(images, conf.KubeSchedulerImage)
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	err = c.PullImages(ctx, c.runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}

	// Configure the etcd
	etcdVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.EtcdImage, "etcd")
	if err != nil {
		return err
	}

	etcdComponentPatches := runtime.GetComponentPatches(config, "etcd")
	etcdComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(etcdComponentPatches.ExtraVolumes)
	if err != nil {
		return fmt.Errorf("failed to expand host volumes for etcd component: %w", err)
	}
	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Workdir:      workdir,
		Image:        conf.EtcdImage,
		Version:      etcdVersion,
		BindAddress:  net.PublicAddress,
		Port:         conf.EtcdPort,
		DataPath:     etcdDataPath,
		Verbosity:    verbosity,
		ExtraArgs:    etcdComponentPatches.ExtraArgs,
		ExtraVolumes: etcdComponentPatches.ExtraVolumes,
		ExtraEnvs:    etcdComponentPatches.ExtraEnvs,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, etcdComponent)

	// Configure the kube-apiserver
	kubeApiserverVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeApiserverImage, "kube-apiserver")
	if err != nil {
		return err
	}

	kubeApiserverComponentPatches := runtime.GetComponentPatches(config, "kube-apiserver")
	kubeApiserverComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(kubeApiserverComponentPatches.ExtraVolumes)
	if err != nil {
		return fmt.Errorf("failed to expand host volumes for kube api server component: %w", err)
	}
	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Workdir:           workdir,
		Image:             conf.KubeApiserverImage,
		Version:           kubeApiserverVersion,
		BindAddress:       net.PublicAddress,
		Port:              conf.KubeApiserverPort,
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
		EtcdPort:          conf.EtcdPort,
		EtcdAddress:       c.Name() + "-etcd",
		Verbosity:         verbosity,
		DisableQPSLimits:  conf.DisableQPSLimits,
		ExtraArgs:         kubeApiserverComponentPatches.ExtraArgs,
		ExtraVolumes:      kubeApiserverComponentPatches.ExtraVolumes,
		ExtraEnvs:         kubeApiserverComponentPatches.ExtraEnvs,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kubeApiserverComponent)

	// Configure the kube-controller-manager
	if !conf.DisableKubeControllerManager {
		kubeControllerManagerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeControllerManagerImage, "kube-controller-manager")
		if err != nil {
			return err
		}

		kubeControllerManagerComponentPatches := runtime.GetComponentPatches(config, "kube-controller-manager")
		kubeControllerManagerComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(kubeControllerManagerComponentPatches.ExtraVolumes)
		if err != nil {
			return fmt.Errorf("failed to expand host volumes for kube controller manager component: %w", err)
		}
		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Workdir:                            workdir,
			Image:                              conf.KubeControllerManagerImage,
			Version:                            kubeControllerManagerVersion,
			BindAddress:                        net.PublicAddress,
			Port:                               conf.KubeControllerManagerPort,
			SecurePort:                         conf.SecurePort,
			CaCertPath:                         caCertPath,
			AdminCertPath:                      adminCertPath,
			AdminKeyPath:                       adminKeyPath,
			KubeAuthorization:                  conf.KubeAuthorization,
			KubeconfigPath:                     inClusterOnHostKubeconfigPath,
			KubeFeatureGates:                   conf.KubeFeatureGates,
			Verbosity:                          verbosity,
			DisableQPSLimits:                   conf.DisableQPSLimits,
			NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
			NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
			ExtraArgs:                          kubeControllerManagerComponentPatches.ExtraArgs,
			ExtraVolumes:                       kubeControllerManagerComponentPatches.ExtraVolumes,
			ExtraEnvs:                          kubeControllerManagerComponentPatches.ExtraEnvs,
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
			err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, inClusterKubeconfig)
			if err != nil {
				return err
			}
		}

		kubeSchedulerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeSchedulerImage, "kube-scheduler")
		if err != nil {
			return err
		}

		kubeSchedulerComponentPatches := runtime.GetComponentPatches(config, "kube-scheduler")
		kubeSchedulerComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(kubeSchedulerComponentPatches.ExtraVolumes)
		if err != nil {
			return fmt.Errorf("failed to expand host volumes for kube scheduler component: %w", err)
		}
		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Workdir:          workdir,
			Image:            conf.KubeSchedulerImage,
			Version:          kubeSchedulerVersion,
			BindAddress:      net.PublicAddress,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       caCertPath,
			AdminCertPath:    adminCertPath,
			AdminKeyPath:     adminKeyPath,
			ConfigPath:       schedulerConfigPath,
			KubeconfigPath:   inClusterOnHostKubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
			Verbosity:        verbosity,
			DisableQPSLimits: conf.DisableQPSLimits,
			ExtraArgs:        kubeSchedulerComponentPatches.ExtraArgs,
			ExtraVolumes:     kubeSchedulerComponentPatches.ExtraVolumes,
			ExtraEnvs:        kubeSchedulerComponentPatches.ExtraEnvs,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeSchedulerComponent)
	}

	// Configure the kwok-controller
	kwokControllerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	kwokControllerComponentPatches := runtime.GetComponentPatches(config, "kwok-controller")
	kwokControllerComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(kwokControllerComponentPatches.ExtraVolumes)
	if err != nil {
		return fmt.Errorf("failed to expand host volumes for kwok controller component: %w", err)
	}

	logVolumes := runtime.GetLogVolumes(ctx)
	kwokControllerExtraVolumes := kwokControllerComponentPatches.ExtraVolumes
	kwokControllerExtraVolumes = append(kwokControllerExtraVolumes, logVolumes...)

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Workdir:                  workdir,
		Image:                    conf.KwokControllerImage,
		Version:                  kwokControllerVersion,
		BindAddress:              net.PublicAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               kwokConfigPath,
		KubeconfigPath:           inClusterOnHostKubeconfigPath,
		CaCertPath:               caCertPath,
		AdminCertPath:            adminCertPath,
		AdminKeyPath:             adminKeyPath,
		NodeName:                 c.Name() + "-kwok-controller",
		Verbosity:                verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		ExtraArgs:                kwokControllerComponentPatches.ExtraArgs,
		ExtraVolumes:             kwokControllerExtraVolumes,
		ExtraEnvs:                kwokControllerComponentPatches.ExtraEnvs,
	})
	config.Components = append(config.Components, kwokControllerComponent)

	// Configure the prometheus
	if conf.PrometheusPort != 0 {
		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:  c.Name(),
			SecurePort:   conf.SecurePort,
			AdminCrtPath: inClusterAdminCertPath,
			AdminKeyPath: inClusterAdminKeyPath,
			Metrics:      runtime.GetMetrics(ctx),
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

		prometheusVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.PrometheusImage, "")
		if err != nil {
			return err
		}

		prometheusComponentPatches := runtime.GetComponentPatches(config, "prometheus")
		prometheusComponentPatches.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(prometheusComponentPatches.ExtraVolumes)
		if err != nil {
			return fmt.Errorf("failed to expand host volumes for prometheus component: %w", err)
		}
		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Workdir:       workdir,
			Image:         conf.PrometheusImage,
			Version:       prometheusVersion,
			BindAddress:   net.PublicAddress,
			Port:          conf.PrometheusPort,
			ConfigPath:    prometheusConfigPath,
			AdminCertPath: adminCertPath,
			AdminKeyPath:  adminKeyPath,
			Verbosity:     verbosity,
			ExtraArgs:     prometheusComponentPatches.ExtraArgs,
			ExtraVolumes:  prometheusComponentPatches.ExtraVolumes,
			ExtraEnvs:     prometheusComponentPatches.ExtraEnvs,
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

	inClusterKubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + c.Name() + "-kube-apiserver:" + format.String(inClusterPort),
		CACrtPath:    inClusterCaCertPath,
		AdminCrtPath: inClusterAdminCertPath,
		AdminKeyPath: inClusterAdminKeyPath,
	}))
	if err != nil {
		return err
	}

	isSelfCompose := c.isSelfCompose(ctx, true)
	if !isSelfCompose {
		composePath := c.GetWorkdirPath(runtime.ComposeName)
		compose := convertToCompose(c.Name(), conf.BindAddress, config.Components)
		composeData, err := yaml.Marshal(compose)
		if err != nil {
			return err
		}
		err = c.WriteFile(composePath, composeData)
		if err != nil {
			return err
		}
	}

	// Save config
	err = c.WriteFile(kubeconfigPath, kubeconfigData)
	if err != nil {
		return err
	}

	err = c.WriteFile(inClusterOnHostKubeconfigPath, inClusterKubeconfigData)
	if err != nil {
		return err
	}

	err = c.SetConfig(ctx, config)
	if err != nil {
		return err
	}
	err = c.Save(ctx)
	if err != nil {
		return err
	}

	if isSelfCompose {
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
	}
	return nil
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	if c.isSelfCompose(ctx, false) {
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
	}

	err := c.Cluster.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	if c.isSelfCompose(ctx, false) {
		return c.start(ctx)
	}
	return c.upCompose(ctx)
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	if c.isSelfCompose(ctx, false) {
		return c.stop(ctx)
	}
	return c.downCompose(ctx)
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	if c.isSelfCompose(ctx, false) {
		return c.start(ctx)
	}
	return c.startCompose(ctx)
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	if c.isSelfCompose(ctx, false) {
		return c.stop(ctx)
	}
	return c.stopCompose(ctx)
}

func (c *Cluster) start(ctx context.Context) error {
	if c.runtime == consts.RuntimeTypeNerdctl {
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

	if c.runtime == consts.RuntimeTypeNerdctl {
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
	if c.runtime == consts.RuntimeTypeNerdctl {
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
	if c.runtime == consts.RuntimeTypeNerdctl {
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
	}, nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	etcdContainerName := c.Name() + "-etcd"

	// If using versions earlier than v3.4, set `ETCDCTL_API=3` to use v3 API.
	args = append([]string{"exec", "--env=ETCDCTL_API=3", "-i", etcdContainerName, "etcdctl"}, args...)
	return c.Exec(ctx, c.runtime, args...)
}

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	for _, component := range config.Components {
		if running, _ := c.inspectComponent(ctx, component.Name); !running {
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
