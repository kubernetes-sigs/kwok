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
	"time"

	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/log"
)

type Cluster struct {
	*runtime.Cluster
}

func NewCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
	}, nil
}

func (c *Cluster) Install(ctx context.Context) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	prometheusPath := ""
	inClusterOnHostKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	composePath := c.GetWorkdirPath(runtime.ComposeName)
	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		err = utils.CreateFile(auditLogPath, 0644)
		if err != nil {
			return err
		}

		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
		err = utils.CopyFile(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	configPath := c.GetWorkdirPath(runtime.ConfigName)

	inClusterConfigPath := "/etc/kwok/kwok.yaml"

	caCertPath := ""
	adminKeyPath := ""
	adminCertPath := ""
	inClusterKubeconfigPath := "/root/.kube/config"
	inClusterEtcdDataPath := "/etcd-data"
	InClusterPrometheusPath := "/etc/prometheus/prometheus.yml"
	inClusterAdminKeyPath := ""
	inClusterAdminCertPath := ""
	inClusterCACertPath := ""
	inClusterPort := uint32(8080)
	scheme := "http"

	// generate ca cert
	if conf.SecurePort {
		err := pki.GeneratePki(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
		caCertPath = utils.PathJoin(pkiPath, "ca.crt")
		adminKeyPath = utils.PathJoin(pkiPath, "admin.key")
		adminCertPath = utils.PathJoin(pkiPath, "admin.crt")
		inClusterPkiPath := "/etc/kubernetes/pki/"
		inClusterCACertPath = utils.PathJoin(inClusterPkiPath, "ca.crt")
		inClusterAdminKeyPath = utils.PathJoin(inClusterPkiPath, "admin.key")
		inClusterAdminCertPath = utils.PathJoin(inClusterPkiPath, "admin.crt")
		inClusterPort = 6443
		scheme = "https"
	}

	// Setup prometheus
	if conf.PrometheusPort != 0 {
		prometheusPath = c.GetWorkdirPath(runtime.Prometheus)
		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:  c.Name(),
			SecurePort:   conf.SecurePort,
			AdminCrtPath: inClusterAdminCertPath,
			AdminKeyPath: inClusterAdminKeyPath,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		err = os.WriteFile(prometheusPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}
	}

	kubeApiserverPort := conf.KubeApiserverPort
	if kubeApiserverPort == 0 {
		kubeApiserverPort, err = utils.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
	}

	// Setup compose
	compose, err := BuildCompose(BuildComposeConfig{
		ProjectName:                  c.Name(),
		KubeApiserverPort:            kubeApiserverPort,
		KubeconfigPath:               inClusterOnHostKubeconfigPath,
		AdminCertPath:                adminCertPath,
		AdminKeyPath:                 adminKeyPath,
		CACertPath:                   caCertPath,
		InClusterKubeconfigPath:      inClusterKubeconfigPath,
		InClusterAdminCertPath:       inClusterAdminCertPath,
		InClusterAdminKeyPath:        inClusterAdminKeyPath,
		InClusterCACertPath:          inClusterCACertPath,
		InClusterEtcdDataPath:        inClusterEtcdDataPath,
		InClusterPrometheusPath:      InClusterPrometheusPath,
		PrometheusPath:               prometheusPath,
		EtcdImage:                    conf.EtcdImage,
		KubeApiserverImage:           conf.KubeApiserverImage,
		KubeControllerManagerImage:   conf.KubeControllerManagerImage,
		KubeSchedulerImage:           conf.KubeSchedulerImage,
		KwokControllerImage:          conf.KwokControllerImage,
		PrometheusImage:              conf.PrometheusImage,
		SecurePort:                   conf.SecurePort,
		Authorization:                conf.KubeAuthorization,
		QuietPull:                    conf.QuietPull,
		DisableKubeScheduler:         conf.DisableKubeScheduler,
		DisableKubeControllerManager: conf.DisableKubeControllerManager,
		PrometheusPort:               conf.PrometheusPort,
		RuntimeConfig:                conf.KubeRuntimeConfig,
		FeatureGates:                 conf.KubeFeatureGates,
		AuditPolicy:                  auditPolicyPath,
		AuditLog:                     auditLogPath,
		ConfigPath:                   configPath,
		InClusterConfigPath:          inClusterConfigPath,
	})
	if err != nil {
		return err
	}

	// Setup kubeconfig
	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://127.0.0.1:" + utils.StringUint32(kubeApiserverPort),
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}
	inClusterKubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + c.Name() + "-kube-apiserver:" + utils.StringUint32(inClusterPort),
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

	err = os.WriteFile(composePath, []byte(compose), 0644)
	if err != nil {
		return err
	}

	// set the context in default kubeconfig
	_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+c.Name()+".server", scheme+"://127.0.0.1:"+utils.StringUint32(kubeApiserverPort))
	_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+c.Name()+".cluster", c.Name())
	if conf.SecurePort {
		_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+c.Name()+".insecure-skip-tls-verify", "true")
		_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+c.Name()+".user", c.Name())
		_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+c.Name()+".client-certificate", adminCertPath)
		_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+c.Name()+".client-key", adminKeyPath)
	}

	images := []string{
		conf.EtcdImage,
		conf.KubeApiserverImage,
		conf.KubeControllerManagerImage,
		conf.KubeSchedulerImage,
		conf.KwokControllerImage,
	}
	if conf.PrometheusPort != 0 {
		images = append(images, conf.PrometheusImage)
	}
	err = runtime.PullImages(ctx, conf.Runtime, images, conf.QuietPull)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Uninstall(ctx context.Context) error {
	// unset the context in default kubeconfig
	_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "clusters."+c.Name())
	_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "users."+c.Name())
	_ = c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+c.Name())

	err := c.Cluster.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	args := []string{"up", "-d"}
	if conf.QuietPull {
		args = append(args, "--quiet-pull")
	}

	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	for i := 0; i != 5; i++ {
		if i != 0 {
			logger.Warn("Try again later", "times", i, "err", err)
			time.Sleep(time.Second)
		}
		err = utils.Exec(ctx, c.Workdir(), utils.IOStreams{
			ErrOut: os.Stderr,
			Out:    os.Stderr,
		}, commands[0], commands[1:]...)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	err = c.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for kube-apiserver ready: %w", err)
	}

	return nil
}

func (c *Cluster) Down(ctx context.Context) error {
	logger := log.FromContext(ctx)
	args := []string{"down"}
	commands, err := c.buildComposeCommands(ctx, args...)
	if err != nil {
		return err
	}

	err = utils.Exec(ctx, c.Workdir(), utils.IOStreams{
		ErrOut: os.Stderr,
		Out:    os.Stderr,
	}, commands[0], commands[1:]...)
	if err != nil {
		logger.Error("Failed to down cluster", err)
	}
	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, c.Workdir(), utils.IOStreams{}, conf.Runtime, "start", c.Name()+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	err = utils.Exec(ctx, c.Workdir(), utils.IOStreams{}, conf.Runtime, "stop", c.Name()+"-"+name)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, c.Name()+"-"+name)
	err = utils.Exec(ctx, c.Workdir(), utils.IOStreams{
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
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	return []string{
		conf.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context) ([]string, error) {
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
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
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	runtime := conf.Runtime
	if runtime == "docker" {
		err := utils.Exec(ctx, "", utils.IOStreams{}, runtime, "compose", "version")
		if err != nil {
			// docker compose subcommand does not exist, try to download it
			dockerComposePath := c.GetBinPath("docker-compose" + conf.BinSuffix)
			err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.DockerComposeBinary, dockerComposePath, 0755, conf.QuietPull)
			if err != nil {
				return nil, err
			}
			return append([]string{dockerComposePath}, args...), nil
		}
	}
	return append([]string{runtime, "compose"}, args...), nil
}

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	etcdContainerName := c.Name() + "-etcd"

	return utils.Exec(ctx, "", stm, conf.Runtime, append([]string{"exec", "-i", etcdContainerName, "etcdctl"}, args...)...)
}
