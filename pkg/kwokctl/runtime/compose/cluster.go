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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
)

// Cluster is an implementation of Runtime for docker.
type Cluster struct {
	*runtime.Cluster

	runtime string

	isNerdctl               bool
	canNerdctlUnlessStopped *bool
	isHostNetwork           bool
}

// NewDockerCluster creates a new Runtime for docker.
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
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
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeNerdctl,
		isNerdctl: true,
	}, nil
}

// NewLimaCluster creates a new Runtime for lima.
func NewLimaCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeNerdctl + "." + consts.RuntimeTypeLima,
		isNerdctl: true,
	}, nil
}

// NewFinchCluster creates a new Runtime for finch.
func NewFinchCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:   runtime.NewCluster(name, workdir),
		runtime:   consts.RuntimeTypeFinch,
		isNerdctl: true,
	}, nil
}

// NewDockerHostCluster creates a new Runtime for docker with host networking.
func NewDockerHostCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster:       runtime.NewCluster(name, workdir),
		runtime:       consts.RuntimeTypeDocker,
		isHostNetwork: true,
	}, nil
}

// Available  checks whether the runtime is available.
func (c *Cluster) Available(ctx context.Context) error {
	if c.IsDryRun() {
		return nil
	}
	return c.Exec(ctx, c.runtime, "version")
}

type env struct {
	kwokctlConfig                 *internalversion.KwokctlConfiguration
	components                    []string
	verbosity                     log.Level
	inClusterOnHostKubeconfigPath string
	inClusterKubeconfig           string
	kubeconfigPath                string
	etcdDataPath                  string
	kwokConfigPath                string
	pkiPath                       string
	auditLogPath                  string
	auditPolicyPath               string
	workdir                       string
	caCertPath                    string
	adminKeyPath                  string
	adminCertPath                 string
	inClusterPkiPath              string
	inClusterCaCertPath           string
	inClusterAdminKeyPath         string
	inClusterAdminCertPath        string
	inClusterPort                 uint32
	scheme                        string
	usedPorts                     sets.Sets[uint32]
}

func (c *Cluster) env(ctx context.Context) (*env, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	components, err := c.Components(ctx)
	if err != nil {
		return nil, err
	}

	inClusterOnHostKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
	inClusterKubeconfig := "/etc/kubernetes/kubeconfig.yaml"
	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	auditLogPath := ""
	auditPolicyPath := ""
	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	workdir := c.Workdir()
	caCertPath := utilspath.Join(pkiPath, "ca.crt")
	adminKeyPath := utilspath.Join(pkiPath, "admin.key")
	adminCertPath := utilspath.Join(pkiPath, "admin.crt")
	inClusterPkiPath := "/etc/kubernetes/pki/"
	inClusterCaCertPath := utilspath.Join(inClusterPkiPath, "ca.crt")
	inClusterAdminKeyPath := utilspath.Join(inClusterPkiPath, "admin.key")
	inClusterAdminCertPath := utilspath.Join(inClusterPkiPath, "admin.crt")

	inClusterPort := uint32(8080)
	scheme := "http"
	if config.Options.SecurePort {
		scheme = "https"
		inClusterPort = 6443
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	usedPorts := runtime.GetUsedPorts(ctx)

	return &env{
		kwokctlConfig:                 config,
		components:                    components,
		verbosity:                     verbosity,
		inClusterOnHostKubeconfigPath: inClusterOnHostKubeconfigPath,
		inClusterKubeconfig:           inClusterKubeconfig,
		kubeconfigPath:                kubeconfigPath,
		etcdDataPath:                  etcdDataPath,
		kwokConfigPath:                kwokConfigPath,
		pkiPath:                       pkiPath,
		auditLogPath:                  auditLogPath,
		auditPolicyPath:               auditPolicyPath,
		workdir:                       workdir,
		caCertPath:                    caCertPath,
		adminKeyPath:                  adminKeyPath,
		adminCertPath:                 adminCertPath,
		inClusterPkiPath:              inClusterPkiPath,
		inClusterCaCertPath:           inClusterCaCertPath,
		inClusterAdminKeyPath:         inClusterAdminKeyPath,
		inClusterAdminCertPath:        inClusterAdminCertPath,
		inClusterPort:                 inClusterPort,
		scheme:                        scheme,
		usedPorts:                     usedPorts,
	}, nil
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
		conf.MetricsServerImage,
	}, nil
}
