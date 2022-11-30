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

	"sigs.k8s.io/kwok/pkg/kwokctl/k8s"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/log"

	"github.com/nxadm/tail"
	"golang.org/x/sync/errgroup"
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
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")

	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeApiserverBinary, kubeApiserverPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeControllerManagerBinary, kubeControllerManagerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KubeSchedulerBinary, kubeSchedulerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	kwokControllerPath := utils.PathJoin(bin, "kwok-controller"+vars.BinSuffix)
	err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.KwokControllerBinary, kwokControllerPath, 0755, conf.QuietPull)
	if err != nil {
		return err
	}

	etcdPath := utils.PathJoin(bin, "etcd"+vars.BinSuffix)
	if conf.EtcdBinary == "" {
		err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdPath, "etcd"+vars.BinSuffix, 0755, conf.QuietPull, true)
		if err != nil {
			return err
		}
	} else {
		err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.EtcdBinary, etcdPath, 0755, conf.QuietPull)
		if err != nil {
			return err
		}
	}

	if conf.PrometheusPort != 0 {
		prometheusPath := utils.PathJoin(bin, "prometheus"+vars.BinSuffix)
		if conf.PrometheusBinary == "" {
			err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.PrometheusBinaryTar, prometheusPath, "prometheus"+vars.BinSuffix, 0755, conf.QuietPull, true)
			if err != nil {
				return err
			}
		} else {
			err = utils.DownloadWithCache(ctx, conf.CacheDir, conf.PrometheusBinary, prometheusPath, 0755, conf.QuietPull)
			if err != nil {
				return err
			}
		}
	}

	etcdDataPath := utils.PathJoin(conf.Workdir, runtime.EtcdDataDirName)
	os.MkdirAll(etcdDataPath, 0755)

	if conf.SecretPort {
		pkiPath := utils.PathJoin(conf.Workdir, runtime.PkiName)
		err = pki.GeneratePki(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %s", err)
		}
	}

	return nil
}

func (c *Cluster) Up(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	scheme := "http"
	if conf.SecretPort {
		scheme = "https"
	}
	bin := utils.PathJoin(conf.Workdir, "bin")

	localAddress := "127.0.0.1"
	serveAddress := "0.0.0.0"

	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver"+vars.BinSuffix)
	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager"+vars.BinSuffix)
	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler"+vars.BinSuffix)
	kwokControllerPath := utils.PathJoin(bin, "kwok-controller"+vars.BinSuffix)
	etcdPath := utils.PathJoin(bin, "etcd"+vars.BinSuffix)
	etcdDataPath := utils.PathJoin(conf.Workdir, runtime.EtcdDataDirName)
	pkiPath := utils.PathJoin(conf.Workdir, runtime.PkiName)
	caCertPath := utils.PathJoin(pkiPath, "ca.crt")
	adminKeyPath := utils.PathJoin(pkiPath, "admin.key")
	adminCertPath := utils.PathJoin(pkiPath, "admin.crt")
	auditLogPath := ""
	auditPolicyPath := ""
	if conf.AuditPolicy != "" {
		auditLogPath = utils.PathJoin(conf.Workdir, "logs", runtime.AuditLogName)
		err = utils.CreateFile(auditLogPath, 0644)
		if err != nil {
			return err
		}

		auditPolicyPath = utils.PathJoin(conf.Workdir, runtime.AuditPolicyName)
		err = utils.CopyFile(conf.AuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	etcdPeerPort := conf.EtcdPeerPort
	if etcdPeerPort == 0 {
		etcdPeerPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		conf.EtcdPeerPort = etcdPeerPort
	}
	etcdPeerPortStr := utils.StringUint32(etcdPeerPort)

	etcdClientPort := conf.EtcdPort
	if etcdClientPort == 0 {
		etcdClientPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		conf.EtcdPort = etcdClientPort
	}
	etcdClientPortStr := utils.StringUint32(etcdClientPort)

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
	err = utils.ForkExec(ctx, conf.Workdir, etcdPath, etcdArgs...)
	if err != nil {
		return err
	}

	kubeApiserverPort := conf.KubeApiserverPort
	if kubeApiserverPort == 0 {
		kubeApiserverPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		conf.KubeApiserverPort = kubeApiserverPort
	}
	kubeApiserverPortStr := utils.StringUint32(kubeApiserverPort)

	kubeApiserverArgs := []string{
		"--admission-control",
		"",
		"--etcd-servers",
		"http://" + localAddress + ":" + etcdClientPortStr,
		"--etcd-prefix",
		"/prefix/registry",
		"--allow-privileged",
	}
	if conf.RuntimeConfig != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--runtime-config",
			conf.RuntimeConfig,
		)
	}
	if conf.FeatureGates != "" {
		kubeApiserverArgs = append(kubeApiserverArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	if conf.SecretPort {
		if conf.Authorization {
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
			utils.PathJoin(conf.Workdir, "cert"),
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

	err = utils.ForkExec(ctx, conf.Workdir, kubeApiserverPath, kubeApiserverArgs...)
	if err != nil {
		return err
	}

	kubeconfigData, err := k8s.BuildKubeconfig(k8s.BuildKubeconfigConfig{
		ProjectName:  conf.Name,
		SecretPort:   conf.SecretPort,
		Address:      scheme + "://" + localAddress + ":" + kubeApiserverPortStr,
		AdminCrtPath: adminCertPath,
		AdminKeyPath: adminKeyPath,
	})
	if err != nil {
		return err
	}

	kubeconfigPath := utils.PathJoin(conf.Workdir, runtime.InHostKubeconfigName)
	err = os.WriteFile(kubeconfigPath, []byte(kubeconfigData), 0644)
	if err != nil {
		return err
	}

	err = c.WaitReady(ctx, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait for kube-apiserver ready: %v", err)
	}

	kubeControllerManagerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
	}
	if conf.FeatureGates != "" {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	kubeControllerManagerPort := conf.KubeControllerManagerPort
	if kubeControllerManagerPort == 0 {
		kubeControllerManagerPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		conf.KubeControllerManagerPort = kubeControllerManagerPort
	}
	if conf.PrometheusPort != 0 {
		if conf.SecretPort {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address",
				localAddress,
				"--secure-port",
				utils.StringUint32(kubeControllerManagerPort),
				"--authorization-always-allow-paths",
				"/healthz,/readyz,/livez,/metrics",
			)
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--address",
				localAddress,
				"--port",
				utils.StringUint32(kubeControllerManagerPort),
				"--secure-port",
				"0",
			)
		}
	}

	if conf.Authorization {
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
	if conf.FeatureGates != "" {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--feature-gates",
			conf.FeatureGates,
		)
	}

	kubeSchedulerPort := conf.KubeSchedulerPort
	if kubeSchedulerPort == 0 {
		kubeSchedulerPort, err = utils.GetUnusedPort()
		if err != nil {
			return err
		}
		conf.KubeSchedulerPort = kubeSchedulerPort
	}
	if conf.SecretPort {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--bind-address",
			localAddress,
			"--secure-port",
			utils.StringUint32(kubeSchedulerPort),
			"--authorization-always-allow-paths",
			"/healthz,/readyz,/livez,/metrics",
		)
	} else {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--address",
			localAddress,
			"--port",
			utils.StringUint32(kubeSchedulerPort),
		)
	}

	kwokControllerArgs := []string{
		"--kubeconfig",
		kubeconfigPath,
		"--manage-all-nodes",
	}

	kwokControllerPort := conf.KwokControllerPort
	if conf.PrometheusPort != 0 {
		if kwokControllerPort == 0 {
			kwokControllerPort, err = utils.GetUnusedPort()
			if err != nil {
				return err
			}
			conf.KwokControllerPort = kwokControllerPort
		}
		kwokControllerArgs = append(kwokControllerArgs,
			"--server-address",
			localAddress+":"+utils.StringUint32(kwokControllerPort),
		)
	}

	var prometheusPath string
	var prometheusArgs []string
	if conf.PrometheusPort != 0 {
		prometheusPortStr := utils.StringUint32(conf.PrometheusPort)

		prometheusData, err := BuildPrometheus(BuildPrometheusConfig{
			ProjectName:               conf.Name,
			SecretPort:                conf.SecretPort,
			AdminCrtPath:              adminCertPath,
			AdminKeyPath:              adminKeyPath,
			PrometheusPort:            conf.PrometheusPort,
			EtcdPort:                  etcdClientPort,
			KubeApiserverPort:         kubeApiserverPort,
			KubeControllerManagerPort: kubeControllerManagerPort,
			KubeSchedulerPort:         kubeSchedulerPort,
			KwokControllerPort:        kwokControllerPort,
		})
		if err != nil {
			return fmt.Errorf("failed to generate prometheus yaml: %s", err)
		}
		prometheusConfigPath := utils.PathJoin(conf.Workdir, runtime.Prometheus)
		err = os.WriteFile(prometheusConfigPath, []byte(prometheusData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write prometheus yaml: %s", err)
		}

		prometheusPath = utils.PathJoin(bin, "prometheus")
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
			return utils.ForkExec(ctx, conf.Workdir, path, args...)
		})
	}
	if err = g.Wait(); err != nil {
		return err
	}

	// set the context in default kubeconfig
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".server", scheme+"://"+localAddress+":"+kubeApiserverPortStr)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".cluster", conf.Name)
	if conf.SecretPort {
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "clusters."+conf.Name+".insecure-skip-tls-verify", "true")
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "contexts."+conf.Name+".user", conf.Name)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-certificate", adminCertPath)
		c.Kubectl(ctx, utils.IOStreams{}, "config", "set", "users."+conf.Name+".client-key", adminKeyPath)
	}

	logger := log.FromContext(ctx)
	err = c.Update(ctx, conf)
	if err != nil {
		logger.Error("Failed to update cluster", err)
	}
	return nil
}

func (c *Cluster) Down(ctx context.Context) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "clusters."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "users."+conf.Name)
	c.Kubectl(ctx, utils.IOStreams{}, "config", "unset", "contexts."+conf.Name)

	bin := utils.PathJoin(conf.Workdir, "bin")
	kubeApiserverPath := utils.PathJoin(bin, "kube-apiserver"+vars.BinSuffix)
	kubeControllerManagerPath := utils.PathJoin(bin, "kube-controller-manager"+vars.BinSuffix)
	kubeSchedulerPath := utils.PathJoin(bin, "kube-scheduler"+vars.BinSuffix)
	kwokControllerPath := utils.PathJoin(bin, "kwok-controller"+vars.BinSuffix)
	etcdPath := utils.PathJoin(bin, "etcd")
	prometheusPath := utils.PathJoin(bin, "prometheus")

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
			err = utils.ForkExecKill(ctx, conf.Workdir, path)
			if err != nil {
				logger.Error("Failed to kill", err,
					"component", filepath.Base(path),
				)
			}
		}(path)
	}
	wg.Wait()

	err = utils.ForkExecKill(ctx, conf.Workdir, kubeApiserverPath)
	if err != nil {
		logger.Error("Failed to kill", err,
			"component", "kube-apiserver",
		)
	}

	err = utils.ForkExecKill(ctx, conf.Workdir, etcdPath)
	if err != nil {
		logger.Error("Failed to kill", err,
			"component", "etcd",
		)
	}

	return nil
}

func (c *Cluster) Start(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")
	svc := utils.PathJoin(bin, name)

	err = utils.ForkExecRestart(ctx, conf.Workdir, svc)
	if err != nil {
		return fmt.Errorf("failed to restart %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Stop(ctx context.Context, name string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")
	svc := utils.PathJoin(bin, name)

	err = utils.ForkExecKill(ctx, conf.Workdir, svc)
	if err != nil {
		return fmt.Errorf("failed to kill %s: %w", name, err)
	}
	return nil
}

func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", filepath.Base(name)+".log")

	f, err := os.OpenFile(logs, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	io.Copy(out, f)
	return nil
}

func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	logs := utils.PathJoin(conf.Workdir, "logs", filepath.Base(name)+".log")

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	defer t.Stop()

	go func() {
		for line := range t.Lines {
			out.Write([]byte(line.Text + "\n"))
		}
	}()
	<-ctx.Done()
	return nil
}

// ListBinaries list binaries in the cluster
func (c *Cluster) ListBinaries(ctx context.Context, actual bool) ([]string, error) {
	if !actual {
		return []string{
			vars.EtcdBinaryTar,
			vars.KubeApiserverBinary,
			vars.KubeControllerManagerBinary,
			vars.KubeSchedulerBinary,
			vars.KwokControllerBinary,
			vars.PrometheusBinaryTar,
			vars.KubectlBinary,
		}, nil
	}
	conf, err := c.Config()
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
		vars.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context, actual bool) ([]string, error) {
	if !actual {
		return []string{}, nil
	}
	_, err := c.Config()
	if err != nil {
		return nil, err
	}
	return []string{}, nil
}
