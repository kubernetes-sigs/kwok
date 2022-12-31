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
	"strings"
	"time"

	"sigs.k8s.io/yaml"

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
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

func (c *Cluster) setup(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	if conf.SecurePort {
		pkiPath := c.GetWorkdirPath(runtime.PkiName)
		if !file.Exists(pkiPath) {
			err = pki.GeneratePki(pkiPath)
			if err != nil {
				return fmt.Errorf("failed to generate pki: %w", err)
			}
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
	err = os.MkdirAll(etcdDataPath, 0755)
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

	err = c.setup(ctx)
	if err != nil {
		return err
	}

	inClusterOnHostKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
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
	inClusterAdminKeyPath := ""
	inClusterAdminCertPath := ""
	inClusterPort := uint32(8080)
	scheme := "http"

	// generate ca cert
	if conf.SecurePort {
		caCertPath = path.Join(pkiPath, "ca.crt")
		adminKeyPath = path.Join(pkiPath, "admin.key")
		adminCertPath = path.Join(pkiPath, "admin.crt")
		inClusterPkiPath := "/etc/kubernetes/pki/"
		inClusterAdminKeyPath = path.Join(inClusterPkiPath, "admin.key")
		inClusterAdminCertPath = path.Join(inClusterPkiPath, "admin.crt")
		scheme = "https"
		inClusterPort = 6443
	}

	err = c.setupPorts(ctx,
		&conf.KubeApiserverPort,
	)
	if err != nil {
		return err
	}

	// Configure the etcd
	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Workdir:  workdir,
		Image:    conf.EtcdImage,
		Port:     conf.EtcdPort,
		DataPath: etcdDataPath,
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, etcdComponent)

	// Configure the kube-apiserver
	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Workdir:           workdir,
		Image:             conf.KubeApiserverImage,
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
	})
	if err != nil {
		return err
	}
	config.Components = append(config.Components, kubeApiserverComponent)

	// Configure the kube-controller-manager
	if !conf.DisableKubeControllerManager {
		kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
			Image:             conf.KubeControllerManagerImage,
			Workdir:           workdir,
			Port:              conf.KubeControllerManagerPort,
			SecurePort:        conf.SecurePort,
			CaCertPath:        caCertPath,
			AdminCertPath:     adminCertPath,
			AdminKeyPath:      adminKeyPath,
			KubeAuthorization: conf.KubeAuthorization,
			KubeconfigPath:    inClusterOnHostKubeconfigPath,
			KubeFeatureGates:  conf.KubeFeatureGates,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeControllerManagerComponent)
	}

	// Configure the kube-scheduler
	if !conf.DisableKubeScheduler {
		kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
			Image:            conf.KubeSchedulerImage,
			Workdir:          workdir,
			Port:             conf.KubeSchedulerPort,
			SecurePort:       conf.SecurePort,
			CaCertPath:       caCertPath,
			AdminCertPath:    adminCertPath,
			AdminKeyPath:     adminKeyPath,
			KubeconfigPath:   inClusterOnHostKubeconfigPath,
			KubeFeatureGates: conf.KubeFeatureGates,
		})
		if err != nil {
			return err
		}
		config.Components = append(config.Components, kubeSchedulerComponent)
	}

	// Configure the kwok-controller
	kwokControllerComponent, err := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Image:          conf.KwokControllerImage,
		Workdir:        workdir,
		Port:           conf.KwokControllerPort,
		ConfigPath:     kwokConfigPath,
		KubeconfigPath: inClusterOnHostKubeconfigPath,
		AdminCertPath:  adminCertPath,
		AdminKeyPath:   adminKeyPath,
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
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}

		prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
			Image:         conf.PrometheusImage,
			Workdir:       workdir,
			Port:          conf.PrometheusPort,
			ConfigPath:    prometheusConfigPath,
			AdminCertPath: adminCertPath,
			AdminKeyPath:  adminKeyPath,
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
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(inClusterOnHostKubeconfigPath, []byte(inClusterKubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(composePath, composeData, 0644)
	if err != nil {
		return err
	}

	// set the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".server", scheme+"://127.0.0.1:"+format.String(conf.KubeApiserverPort))
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".cluster", c.Name())
	if conf.SecurePort {
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".insecure-skip-tls-verify", "true")
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".user", c.Name())
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-certificate", adminCertPath)
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-key", adminKeyPath)
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
	err = image.PullImages(ctx, conf.Runtime, images, conf.QuietPull)
	if err != nil {
		return err
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
		err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{
			ErrOut: os.Stderr,
			Out:    os.Stderr,
		}, commands[0], commands[1:]...)
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

func (c *Cluster) isRunning(ctx context.Context) (bool, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	commands, err := c.buildComposeCommands(ctx, "ps")
	if err != nil {
		return false, err
	}
	out := bytes.NewBuffer(nil)
	err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{
		Out: out,
	}, commands[0], commands[1:]...)
	if err != nil {
		return false, err
	}
	i := strings.Count(out.String(), "running")
	if i != len(config.Components) {
		logger := log.FromContext(ctx)
		logger.Debug("Components not all running",
			"running", i,
			"total", len(config.Components),
			"output", out.String(),
		)
		return false, nil
	}
	return true, nil
}

func (c *Cluster) Down(ctx context.Context) error {
	logger := log.FromContext(ctx)
	args := []string{"down"}
	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{
		ErrOut: os.Stderr,
		Out:    os.Stderr,
	}, commands[0], commands[1:]...)
	if err != nil {
		logger.Error("Failed to down cluster", err)
	}
	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{}, conf.Runtime, "start", c.Name()+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{}, conf.Runtime, "stop", c.Name()+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, c.Name()+"-"+name)
	err = exec.Exec(ctx, c.Workdir(), exec.IOStreams{
		ErrOut: out,
		Out:    out,
	}, conf.Runtime, args...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, false)
}

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

	runtime := conf.Runtime
	if runtime == "docker" {
		err := exec.Exec(ctx, "", exec.IOStreams{}, runtime, "compose", "version")
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
	return append([]string{runtime, "compose"}, args...), nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, stm exec.IOStreams, args ...string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	etcdContainerName := c.Name() + "-etcd"

	return exec.Exec(ctx, "", stm, conf.Runtime, append([]string{"exec", "-i", etcdContainerName, "etcdctl"}, args...)...)
}
