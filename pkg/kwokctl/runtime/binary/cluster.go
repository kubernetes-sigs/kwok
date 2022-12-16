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
	"sync"
	"time"

	"github.com/nxadm/tail"
	"golang.org/x/sync/errgroup"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
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

func (c *Cluster) Install(ctx context.Context) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

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

	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	err = os.MkdirAll(etcdDataPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to mkdir etcd data path: %w", err)
	}

	if conf.SecurePort {
		pkiPath := c.GetWorkdirPath(runtime.PkiName)
		err = pki.GeneratePki(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	scheme := "http"
	if conf.SecurePort {
		scheme = "https"
	}

	localAddress := "127.0.0.1"
	serveAddress := "0.0.0.0"

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	caCertPath := path.Join(pkiPath, "ca.crt")
	adminKeyPath := path.Join(pkiPath, "admin.key")
	adminCertPath := path.Join(pkiPath, "admin.crt")

	etcdClientPort := conf.EtcdPort
	if etcdClientPort == 0 {
		etcdClientPort, err = net.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
		conf.EtcdPort = etcdClientPort
	}
	etcdClientPortStr := format.String(etcdClientPort)

	if err := c.setupEtcd(ctx, conf, localAddress, etcdClientPortStr); err != nil {
		return err
	}

	kubeApiserverPort := conf.KubeApiserverPort
	if kubeApiserverPort == 0 {
		kubeApiserverPort, err = net.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
		conf.KubeApiserverPort = kubeApiserverPort
	}
	kubeApiserverPortStr := format.String(kubeApiserverPort)

	if err := c.setupApiserver(ctx, conf, localAddress, serveAddress, etcdClientPortStr, kubeApiserverPortStr, caCertPath, adminKeyPath, adminCertPath); err != nil {
		return err
	}

	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      scheme + "://" + localAddress + ":" + kubeApiserverPortStr,
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = c.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for kube-apiserver ready: %w", err)
	}

	if err := c.setupComponents(ctx, conf, localAddress, serveAddress, kubeconfigPath, caCertPath, adminKeyPath, adminCertPath); err != nil {
		return err
	}

	// set the context in default kubeconfig
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".server", scheme+"://"+localAddress+":"+kubeApiserverPortStr)
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".cluster", c.Name())
	if conf.SecurePort {
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "clusters."+c.Name()+".insecure-skip-tls-verify", "true")
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "contexts."+c.Name()+".user", c.Name())
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-certificate", adminCertPath)
		_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "set", "users."+c.Name()+".client-key", adminKeyPath)
	}

	logger := log.FromContext(ctx)
	err = c.SetConfig(ctx, conf)
	if err != nil {
		logger.Error("Failed to set config", err)
	}
	err = c.Save(ctx)
	if err != nil {
		logger.Error("Failed to update cluster", err)
	}
	return nil
}

func (c *Cluster) setupEtcd(ctx context.Context, conf *internalversion.KwokctlConfigurationOptions, localAddress string, etcdClientPortStr string) error {
	var err error
	etcdPath := c.GetBinPath("etcd" + conf.BinSuffix)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	etcdPeerPort := conf.EtcdPeerPort
	if etcdPeerPort == 0 {
		etcdPeerPort, err = net.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
		conf.EtcdPeerPort = etcdPeerPort
	}
	etcdPeerPortStr := format.String(etcdPeerPort)

	etcdArgs := []string{
		"--data-dir",
		etcdDataPath,
		"--name",
		"node0",
		"--initial-advertise-peer-urls",
		"http://" + localAddress + ":" + etcdPeerPortStr,
		"--listen-peer-urls",
		"http://" + localAddress + ":" + etcdPeerPortStr,
		"--advertise-client-urls",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--listen-client-urls",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--initial-cluster",
		"node0=http://" + localAddress + ":" + etcdPeerPortStr,
		"--auto-compaction-retention",
		"1",
		"--quota-backend-bytes",
		"8589934592",
	}

	return exec.ForkExec(ctx, c.Workdir(), etcdPath, etcdArgs...)
}

func (c *Cluster) setupApiserver(ctx context.Context, conf *internalversion.KwokctlConfigurationOptions, localAddress, serveAddress, etcdClientPortStr, kubeApiserverPortStr, caCertPath, adminKeyPath, adminCertPath string) error {
	var err error
	kubeApiserverPath := c.GetBinPath("kube-apiserver" + conf.BinSuffix)
	auditLogPath := ""
	auditPolicyPath := ""
	if conf.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		err = file.Create(auditLogPath, 0644)
		if err != nil {
			return err
		}

		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
		err = file.Copy(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	kubeApiserverArgs := []string{
		"--admission-control",
		"",
		"--etcd-servers",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--etcd-prefix",
		"/registry",
		"--allow-privileged",
	}
	if conf.KubeRuntimeConfig != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--runtime-config",
			conf.KubeRuntimeConfig,
		)
	}
	if conf.KubeFeatureGates != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--feature-gates",
			conf.KubeFeatureGates,
		)
	}

	if conf.SecurePort {
		if conf.KubeAuthorization {
			kubeApiserverArgs = append(kubeApiserverArgs,
				"--authorization-mode",
				"Node,RBAC",
			)
		}
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--bind-address",
			serveAddress,
			"--secure-port",
			kubeApiserverPortStr,
			"--tls-cert-file",
			adminCertPath,
			"--tls-private-key-file",
			adminKeyPath,
			"--client-ca-file",
			caCertPath,
			"--service-account-key-file",
			adminKeyPath,
			"--service-account-signing-key-file",
			adminKeyPath,
			"--service-account-issuer",
			"https://kubernetes.default.svc.cluster.local",
		)
	} else {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--insecure-bind-address",
			serveAddress,
			"--insecure-port",
			kubeApiserverPortStr,
			"--cert-dir",
			c.GetWorkdirPath("cert"),
		)
	}

	if auditPolicyPath != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--audit-policy-file",
			auditPolicyPath,
			"--audit-log-path",
			auditLogPath,
		)
	}

	return exec.ForkExec(ctx, c.Workdir(), kubeApiserverPath, kubeApiserverArgs...)
}

func (c *Cluster) setupComponents(ctx context.Context, conf *internalversion.KwokctlConfigurationOptions, localAddress, serveAddress, kubeconfigPath, caCertPath, adminKeyPath, adminCertPath string) error {
	kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
	kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)

	var err error
	kubeControllerManagerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	if conf.KubeFeatureGates != "" {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--feature-gates",
			conf.KubeFeatureGates,
		)
	}

	kubeControllerManagerPort := conf.KubeControllerManagerPort
	if kubeControllerManagerPort == 0 {
		kubeControllerManagerPort, err = net.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
		conf.KubeControllerManagerPort = kubeControllerManagerPort
	}
	if conf.PrometheusPort != 0 {
		if conf.SecurePort {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address",
				localAddress,
				"--secure-port",
				format.String(kubeControllerManagerPort),
				"--authorization-always-allow-paths",
				"/healthz,/readyz,/livez,/metrics",
			)
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--address",
				localAddress,
				"--port",
				format.String(kubeControllerManagerPort),
				"--secure-port",
				"0",
			)
		}
	}

	if conf.KubeAuthorization {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--root-ca-file",
			caCertPath,
			"--service-account-private-key-file",
			adminKeyPath,
		)
	}

	kubeSchedulerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	if conf.KubeFeatureGates != "" {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--feature-gates",
			conf.KubeFeatureGates,
		)
	}

	kubeSchedulerPort := conf.KubeSchedulerPort
	if kubeSchedulerPort == 0 {
		kubeSchedulerPort, err = net.GetUnusedPort(ctx)
		if err != nil {
			return err
		}
		conf.KubeSchedulerPort = kubeSchedulerPort
	}
	if conf.SecurePort {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--bind-address",
			localAddress,
			"--secure-port",
			format.String(kubeSchedulerPort),
			"--authorization-always-allow-paths",
			"/healthz,/readyz,/livez,/metrics",
		)
	} else {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--address",
			localAddress,
			"--port",
			format.String(kubeSchedulerPort),
		)
	}

	configPath := c.GetWorkdirPath(runtime.ConfigName)
	kwokControllerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
		"--config",
		configPath,
		"--manage-all-nodes",
	}

	kwokControllerPort := conf.KwokControllerPort
	if conf.PrometheusPort != 0 {
		if kwokControllerPort == 0 {
			kwokControllerPort, err = net.GetUnusedPort(ctx)
			if err != nil {
				return err
			}
			conf.KwokControllerPort = kwokControllerPort
		}
		kwokControllerArgs = append(kwokControllerArgs,
			"--server-address",
			localAddress+":"+format.String(kwokControllerPort),
		)
	}

	var prometheusPath string
	var prometheusArgs []string
	if conf.PrometheusPort != 0 {
		prometheusPortStr := format.String(conf.PrometheusPort)

		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:               c.Name(),
			SecurePort:                conf.SecurePort,
			AdminCrtPath:              adminCertPath,
			AdminKeyPath:              adminKeyPath,
			PrometheusPort:            conf.PrometheusPort,
			EtcdPort:                  conf.EtcdPort,
			KubeApiserverPort:         conf.KubeApiserverPort,
			KubeControllerManagerPort: kubeControllerManagerPort,
			KubeSchedulerPort:         kubeSchedulerPort,
			KwokControllerPort:        kwokControllerPort,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %w", err)
		}
		prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %w", err)
		}

		prometheusPath = c.GetBinPath("prometheus")
		prometheusArgs = []string{
			"--config.file",
			prometheusConfigPath,
			"--web.listen-address",
			serveAddress + ":" + prometheusPortStr,
		}
	}

	componentPathArgs := map[string][]string{
		kwokControllerPath: kwokControllerArgs,
	}
	if !conf.DisableKubeControllerManager {
		componentPathArgs[kubeControllerManagerPath] = kubeControllerManagerArgs
	}
	if !conf.DisableKubeScheduler {
		componentPathArgs[kubeSchedulerPath] = kubeSchedulerArgs
	}
	if prometheusPath != "" {
		componentPathArgs[prometheusPath] = prometheusArgs
	}

	// do not use ctx which returns from errgroup, g.wait() will cancel the ctx and kill the daemon process
	g, _ := errgroup.WithContext(ctx)
	for path, args := range componentPathArgs {
		path, args := path, args
		g.Go(func() error {
			return exec.ForkExec(ctx, c.Workdir(), path, args...)
		})
	}
	return g.Wait()
}

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "clusters."+c.Name())
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "users."+c.Name())
	_ = c.Kubectl(ctx, exec.IOStreams{}, "config", "unset", "contexts."+c.Name())

	kubeApiserverPath := c.GetBinPath("kube-apiserver" + conf.BinSuffix)
	kubeControllerManagerPath := c.GetBinPath("kube-controller-manager" + conf.BinSuffix)
	kubeSchedulerPath := c.GetBinPath("kube-scheduler" + conf.BinSuffix)
	kwokControllerPath := c.GetBinPath("kwok-controller" + conf.BinSuffix)
	etcdPath := c.GetBinPath("etcd")
	prometheusPath := c.GetBinPath("prometheus")

	componentPaths := []string{
		kwokControllerPath,
	}

	if !conf.DisableKubeControllerManager {
		componentPaths = append(componentPaths, kubeControllerManagerPath)
	}

	if !conf.DisableKubeScheduler {
		componentPaths = append(componentPaths, kubeSchedulerPath)
	}

	if conf.PrometheusPort != 0 {
		componentPaths = append(componentPaths, prometheusPath)
	}

	logger := log.FromContext(ctx)
	var wg sync.WaitGroup
	for _, path := range componentPaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			err = exec.ForkExecKill(ctx, c.Workdir(), path)
			if err != nil {
				logger.Error("Failed to kill", err,
					"component", filepath.Base(path),
				)
			}
		}(path)
	}
	wg.Wait()

	err = exec.ForkExecKill(ctx, c.Workdir(), kubeApiserverPath)
	if err != nil {
		logger.Error("Failed to kill", err,
			"component", "kube-apiserver",
		)
	}

	err = exec.ForkExecKill(ctx, c.Workdir(), etcdPath)
	if err != nil {
		logger.Error("Failed to kill", err,
			"component", "etcd",
		)
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	svc := c.GetBinPath(name)

	err := exec.ForkExecRestart(ctx, c.Workdir(), svc)
	if err != nil {
		return fmt.Errorf("failed to restart %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	svc := c.GetBinPath(name)

	err := exec.ForkExecKill(ctx, c.Workdir(), svc)
	if err != nil {
		return fmt.Errorf("failed to kill %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	logger := log.FromContext(ctx)

	logs := c.GetLogPath(filepath.Base(name) + ".log")

	f, err := os.OpenFile(logs, os.O_RDONLY, 0644)
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
	conf, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

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
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	etcdctlPath := c.GetBinPath("etcdctl" + conf.BinSuffix)

	err = file.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdctlPath, "etcdctl"+conf.BinSuffix, 0755, conf.QuietPull, true)
	if err != nil {
		return err
	}

	return exec.Exec(ctx, "", stm, etcdctlPath, append([]string{"--endpoints", "127.0.0.1:" + format.String(conf.EtcdPort)}, args...)...)
}
