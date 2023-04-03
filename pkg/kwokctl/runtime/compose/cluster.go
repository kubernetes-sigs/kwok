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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/image"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Cluster is an implementation of Runtime for docker-compose.
type Cluster struct {
	*runtime.Cluster

	runtime string
}

// NewNerdctlCluster creates a new Runtime for nerdctl compose.
func NewNerdctlCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeNerdctl,
	}, nil
}

// NewDockerCluster creates a new Runtime for docker-compose.
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	return exec.Exec(ctx, c.runtime, "version")
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
		err = file.Create(auditLogPath, 0640)
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
	composePath := c.GetWorkdirPath(runtime.ComposeName)
	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	workdir := c.Workdir()
	adminKeyPath := ""
	adminCertPath := ""
	caCertPath := ""
	caCertPath = path.Join(pkiPath, "ca.crt")
	adminKeyPath = path.Join(pkiPath, "admin.key")
	adminCertPath = path.Join(pkiPath, "admin.crt")
	inClusterPkiPath := "/etc/kubernetes/pki/"
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
	err = image.PullImages(ctx, c.runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}

	// Configure the etcd
	etcdVersion, err := version.ParseFromImage(ctx, c.runtime, conf.EtcdImage, "etcd")
	if err != nil {
		return err
	}

	etedComponentPatches := runtime.GetComponentPatches(config, "etcd")
	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Workdir:      workdir,
		Image:        conf.EtcdImage,
		Version:      etcdVersion,
		Port:         conf.EtcdPort,
		DataPath:     etcdDataPath,
		ExtraArgs:    etedComponentPatches.ExtraArgs,
		ExtraVolumes: etedComponentPatches.ExtraVolumes,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, etcdComponent)

	// Configure the kube-apiserver
	kubeApiserverVersion, err := version.ParseFromImage(ctx, c.runtime, conf.KubeApiserverImage, "kube-apiserver")
	if err != nil {
		return err
	}

	kubeApiserverComponentPatches := runtime.GetComponentPatches(config, "kube-apiserver")
	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Workdir:           workdir,
		Image:             conf.KubeApiserverImage,
		Version:           kubeApiserverVersion,
		Port:              conf.KubeApiserverPort,
		KubeRuntimeConfig: conf.KubeRuntimeConfig,
		KubeFeatureGates:  conf.KubeFeatureGates,
		SecurePort:        conf.SecurePort,
		KubeAuthorization: conf.KubeAuthorization,
		AuditPolicyPath:   auditPolicyPath,
		AuditLogPath:      auditLogPath,
		CaCertPath:        caCertPath,
		AdminCertPath:     adminCertPath,
		AdminKeyPath:      adminKeyPath,
		EtcdPort:          conf.EtcdPort,
		EtcdAddress:       c.Name() + "-etcd",
		ExtraArgs:         kubeApiserverComponentPatches.ExtraArgs,
		ExtraVolumes:      kubeApiserverComponentPatches.ExtraVolumes,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kubeApiserverComponent)

	// Configure the kube-controller-manager
	if !conf.DisableKubeControllerManager {
		kubeControllerManagerVersion, err := version.ParseFromImage(ctx, c.runtime, conf.KubeControllerManagerImage, "kube-controller-manager")
		if err != nil {
			return err
		}

		kubeControllerManagerComponentPatches := runtime.GetComponentPatches(config, "kube-controller-manager")
		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Workdir:                            workdir,
			Image:                              conf.KubeControllerManagerImage,
			Version:                            kubeControllerManagerVersion,
			Port:                               conf.KubeControllerManagerPort,
			SecurePort:                         conf.SecurePort,
			CaCertPath:                         caCertPath,
			AdminCertPath:                      adminCertPath,
			AdminKeyPath:                       adminKeyPath,
			KubeAuthorization:                  conf.KubeAuthorization,
			KubeconfigPath:                     inClusterOnHostKubeconfigPath,
			KubeFeatureGates:                   conf.KubeFeatureGates,
			NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
			NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
			ExtraArgs:                          kubeControllerManagerComponentPatches.ExtraArgs,
			ExtraVolumes:                       kubeControllerManagerComponentPatches.ExtraVolumes,
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
			err = k8s.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, inClusterKubeconfig)
			if err != nil {
				return err
			}
		}

		kubeSchedulerVersion, err := version.ParseFromImage(ctx, c.runtime, conf.KubeSchedulerImage, "kube-scheduler")
		if err != nil {
			return err
		}

		kubeSchedulerComponentPatches := runtime.GetComponentPatches(config, "kube-scheduler")
		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Workdir:          workdir,
			Image:            conf.KubeSchedulerImage,
			Version:          kubeSchedulerVersion,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       caCertPath,
			AdminCertPath:    adminCertPath,
			AdminKeyPath:     adminKeyPath,
			ConfigPath:       schedulerConfigPath,
			KubeconfigPath:   inClusterOnHostKubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
			ExtraArgs:        kubeSchedulerComponentPatches.ExtraArgs,
			ExtraVolumes:     kubeSchedulerComponentPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeSchedulerComponent)
	}

	// Configure the kwok-controller
	kwokControllerVersion, err := version.ParseFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	kwokControllerComponentPatches := runtime.GetComponentPatches(config, "kwok-controller")
	kwokControllerComponent, err := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Workdir:        workdir,
		Image:          conf.KwokControllerImage,
		Version:        kwokControllerVersion,
		Port:           conf.KwokControllerPort,
		ConfigPath:     kwokConfigPath,
		KubeconfigPath: inClusterOnHostKubeconfigPath,
		AdminCertPath:  adminCertPath,
		AdminKeyPath:   adminKeyPath,
		NodeName:       c.Name() + "-kwok-controller",
		ExtraArgs:      kwokControllerComponentPatches.ExtraArgs,
		ExtraVolumes:   kwokControllerComponentPatches.ExtraVolumes,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kwokControllerComponent)

	// Configure the prometheus
	if conf.PrometheusPort != 0 {
		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:  c.Name(),
			SecurePort:   conf.SecurePort,
			AdminCrtPath: inClusterAdminCertPath,
			AdminKeyPath: inClusterAdminKeyPath,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)
		//nolint:gosec
		// We don't need to check the permissions of the prometheus config file,
		// because it's working in a non-root container.
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}

		prometheusVersion, err := version.ParseFromImage(ctx, c.runtime, conf.PrometheusImage, "")
		if err != nil {
			return err
		}

		prometheusComponentPatches := runtime.GetComponentPatches(config, "prometheus")
		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Workdir:       workdir,
			Image:         conf.PrometheusImage,
			Version:       prometheusVersion,
			Port:          conf.PrometheusPort,
			ConfigPath:    prometheusConfigPath,
			AdminCertPath: adminCertPath,
			AdminKeyPath:  adminKeyPath,
			ExtraArgs:     prometheusComponentPatches.ExtraArgs,
			ExtraVolumes:  prometheusComponentPatches.ExtraVolumes,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, prometheusComponent)
	}

	// Setup compose
	compose := convertToCompose(c.Name(), config.Components)
	composeData, err := yaml.Marshal(compose)
	if err != nil {
		return err
	}

	// Setup kubeconfig
	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://127.0.0.1:" + format.String(conf.KubeApiserverPort),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}

	inClusterKubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + c.Name() + "-kube-apiserver:" + format.String(inClusterPort),
		AdminCrtPath: inClusterAdminCertPath,
		AdminKeyPath: inClusterAdminKeyPath,
	})
	if err != nil {
		return err
	}

	// Save config
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0640)
	if err != nil {
		return err
	}

	err = os.WriteFile(inClusterOnHostKubeconfigPath, []byte(inClusterKubeconfigData), 0640)
	if err != nil {
		return err
	}

	err = os.WriteFile(composePath, composeData, 0640)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
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

// Up starts the cluster.
func (c *Cluster) Up(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	args := []string{"up", "-d"}
	if conf.QuietPull {
		args = append(args, "--quiet-pull")
	}

	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	for i := 0; ctx.Err() == nil; i++ {
		err = exec.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
		if err != nil {
			logger.Debug("Failed to start cluster",
				"times", i,
				"err", err,
			)
			time.Sleep(time.Second)
			continue
		}
		ready, err := c.isRunning(ctx)
		if err != nil {
			logger.Debug("Failed to check components status",
				"times", i,
				"err", err,
			)
			time.Sleep(time.Second)
			continue
		}
		if !ready {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	err = ctx.Err()
	if err != nil {
		return err
	}

	return nil
}

type statusItem struct {
	Service string
	State   string
}

func (c *Cluster) isRunning(ctx context.Context) (bool, error) {
	commands, err := c.buildComposeCommands(ctx, "ps", "--format=json")
	if err != nil {
		return false, err
	}
	out := bytes.NewBuffer(nil)
	err = exec.Exec(exec.WithWriteTo(exec.WithDir(ctx, c.Workdir()), out), commands[0], commands[1:]...)
	if err != nil {
		return false, err
	}

	var data []statusItem
	err = json.Unmarshal(out.Bytes(), &data)
	if err != nil {
		return false, err
	}

	if len(data) == 0 {
		logger := log.FromContext(ctx)
		logger.Debug("No components found")
		return false, nil
	}

	components, ok := slices.Find(data, func(i statusItem) bool {
		return i.State != "running"
	})
	if ok {
		logger := log.FromContext(ctx)
		logger.Debug("Components not all running",
			"components", components,
		)
		return false, nil
	}
	return true, nil
}

// Down stops the cluster
func (c *Cluster) Down(ctx context.Context) error {
	logger := log.FromContext(ctx)
	args := []string{"down"}
	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	err = exec.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		logger.Error("Failed to down cluster", err)
	}
	return nil
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	// TODO: nerdctl does not support 'compose start' in v1.1.0 or earlier
	// Support in https://github.com/containerd/nerdctl/pull/1656 merge into the main branch, but there is no release
	subcommand := []string{"start"}
	if c.runtime == consts.RuntimeTypeNerdctl {
		subcommand = []string{"up", "-d"}
	}

	commands, err := c.buildComposeCommands(ctx, subcommand...)
	if err != nil {
		return err
	}

	err = exec.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		return fmt.Errorf("failed to start cluster: %w", err)
	}

	if c.runtime == consts.RuntimeTypeNerdctl {
		backupFilename := c.GetWorkdirPath("restart.db")
		fi, err := os.Stat(backupFilename)
		if err == nil {
			if fi.IsDir() {
				return fmt.Errorf("wrong backup file %s, it cannot be a directory, please remove it", backupFilename)
			}
			if err := c.SnapshotRestore(ctx, backupFilename); err != nil {
				return fmt.Errorf("failed to restore cluster data: %w", err)
			}
			if err := os.Remove(backupFilename); err != nil {
				return fmt.Errorf("failed to remove backup file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// Stop stops the cluster
func (c *Cluster) Stop(ctx context.Context) error {
	// TODO: nerdctl does not support 'compose stop' in v1.0.0 or earlier
	subcommand := "stop"
	if c.runtime == consts.RuntimeTypeNerdctl {
		subcommand = "down"
		err := c.SnapshotSave(ctx, c.GetWorkdirPath("restart.db"))
		if err != nil {
			return fmt.Errorf("failed to snapshot cluster data: %w", err)
		}
	}

	commands, err := c.buildComposeCommands(ctx, subcommand)
	if err != nil {
		return err
	}

	err = exec.Exec(exec.WithAllWriteToErrOut(exec.WithDir(ctx, c.Workdir())), commands[0], commands[1:]...)
	if err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}
	return nil
}

// StartComponent starts a component in the cluster
func (c *Cluster) StartComponent(ctx context.Context, name string) error {
	return exec.Exec(exec.WithDir(ctx, c.Workdir()), c.runtime, "start", c.Name()+"-"+name)
}

// StopComponent stops a component in the cluster
func (c *Cluster) StopComponent(ctx context.Context, name string) error {
	return exec.Exec(exec.WithDir(ctx, c.Workdir()), c.runtime, "stop", c.Name()+"-"+name)
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, c.Name()+"-"+name)
	err := exec.Exec(exec.WithAllWriteTo(exec.WithDir(ctx, c.Workdir()), out), c.runtime, args...)
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
		conf.EtcdImage,
		conf.KubeApiserverImage,
		conf.KubeControllerManagerImage,
		conf.KubeSchedulerImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
	}, nil
}

// buildComposeCommands returns the compose commands with given current runtime and args
func (c *Cluster) buildComposeCommands(ctx context.Context, args ...string) ([]string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	if c.runtime == consts.RuntimeTypeDocker {
		err := exec.Exec(ctx, c.runtime, "compose", "version")
		if err != nil {
			// docker compose subcommand does not exist, try to download it
			dockerComposePath := c.GetBinPath("docker-compose" + conf.BinSuffix)
			err = file.DownloadWithCache(ctx, conf.CacheDir, conf.DockerComposeBinary, dockerComposePath, 0755, conf.QuietPull)
			if err != nil {
				return nil, err
			}
			return append([]string{dockerComposePath}, args...), nil
		}
	}
	return append([]string{c.runtime, "compose"}, args...), nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	etcdContainerName := c.Name() + "-etcd"
	return exec.Exec(ctx, c.runtime, append([]string{"exec", "-i", etcdContainerName, "etcdctl"}, args...)...)
}
