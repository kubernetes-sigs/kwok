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

package kind

import (
	"context"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
)

// Cluster is an implementation of Runtime for kind
type Cluster struct {
	*runtime.Cluster

	runtime string
}

// NewDockerCluster creates a new Runtime for kind with docker
func NewDockerCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeDocker,
	}, nil
}

// NewPodmanCluster creates a new Runtime for kind with podman
func NewPodmanCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypePodman,
	}, nil
}

// NewNerdctlCluster creates a new Runtime for kind with nerdctl
func NewNerdctlCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeNerdctl,
	}, nil
}

// NewLimaCluster creates a new Runtime for kind with lima
func NewLimaCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeNerdctl + "." + consts.RuntimeTypeLima,
	}, nil
}

// NewFinchCluster creates a new Runtime for kind with finch
func NewFinchCluster(name, workdir string) (runtime.Runtime, error) {
	return &Cluster{
		Cluster: runtime.NewCluster(name, workdir),
		runtime: consts.RuntimeTypeFinch,
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
	kwokctlConfig        *internalversion.KwokctlConfiguration
	components           []string
	verbosity            log.Level
	schedulerConfigPath  string
	auditLogPath         string
	auditPolicyPath      string
	prometheusConfigPath string

	inClusterOnHostKubeconfigPath string
	workdir                       string
	caCertPath                    string
	adminKeyPath                  string
	adminCertPath                 string

	kwokConfigPath string

	usedPorts sets.Sets[uint32]
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

	inClusterOnHostKubeconfigPath := "/etc/kubernetes/admin.conf"
	schedulerConfigPath := "/etc/kubernetes/scheduler.conf"
	prometheusConfigPath := "/etc/prometheus/prometheus.yaml"
	kwokConfigPath := "/etc/kwok/kwok.yaml"
	auditLogPath := ""
	auditPolicyPath := ""
	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	pkiPath := "/etc/kubernetes/pki"

	workdir := c.Workdir()
	caCertPath := utilspath.Join(pkiPath, "ca.crt")
	adminKeyPath := utilspath.Join(pkiPath, "admin.key")
	adminCertPath := utilspath.Join(pkiPath, "admin.crt")

	usedPorts := runtime.GetUsedPorts(ctx)
	return &env{
		kwokctlConfig:                 config,
		components:                    components,
		verbosity:                     verbosity,
		schedulerConfigPath:           schedulerConfigPath,
		prometheusConfigPath:          prometheusConfigPath,
		auditLogPath:                  auditLogPath,
		auditPolicyPath:               auditPolicyPath,
		inClusterOnHostKubeconfigPath: inClusterOnHostKubeconfigPath,
		workdir:                       workdir,
		caCertPath:                    caCertPath,
		adminKeyPath:                  adminKeyPath,
		adminCertPath:                 adminCertPath,
		kwokConfigPath:                kwokConfigPath,
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
		conf.KindNodeImage,
		conf.KwokControllerImage,
		conf.PrometheusImage,
		conf.MetricsServerImage,
	}, nil
}
