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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
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

type env struct {
	kwokctlConfig           *internalversion.KwokctlConfiguration
	components              []string
	verbosity               log.Level
	inClusterKubeconfigPath string
	kubeconfigPath          string
	etcdDataPath            string
	kwokConfigPath          string
	pkiPath                 string
	auditLogPath            string
	auditPolicyPath         string
	workdir                 string
	caCertPath              string
	adminKeyPath            string
	adminCertPath           string
	scheme                  string
	usedPorts               sets.Sets[uint32]
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

	scheme := "http"
	if config.Options.SecurePort {
		scheme = "https"
	}

	workdir := c.Workdir()

	kubeconfigPath := c.GetWorkdirPath(runtime.InHostKubeconfigName)
	inClusterKubeconfigPath := c.GetWorkdirPath(runtime.InClusterKubeconfigName)
	if config.Options.KubeApiserverInsecurePort == 0 {
		inClusterKubeconfigPath = kubeconfigPath
	}

	kwokConfigPath := c.GetWorkdirPath(runtime.ConfigName)
	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	caCertPath := utilspath.Join(pkiPath, "ca.crt")
	adminKeyPath := utilspath.Join(pkiPath, "admin.key")
	adminCertPath := utilspath.Join(pkiPath, "admin.crt")
	auditLogPath := ""
	auditPolicyPath := ""

	if config.Options.KubeAuditPolicy != "" {
		auditLogPath = c.GetLogPath(runtime.AuditLogName)
		auditPolicyPath = c.GetWorkdirPath(runtime.AuditPolicyName)
	}

	logger := log.FromContext(ctx)
	verbosity := logger.Level()

	usedPorts := runtime.GetUsedPorts(ctx)

	return &env{
		kwokctlConfig:           config,
		components:              components,
		verbosity:               verbosity,
		inClusterKubeconfigPath: inClusterKubeconfigPath,
		kubeconfigPath:          kubeconfigPath,
		etcdDataPath:            etcdDataPath,
		kwokConfigPath:          kwokConfigPath,
		pkiPath:                 pkiPath,
		auditLogPath:            auditLogPath,
		auditPolicyPath:         auditPolicyPath,
		workdir:                 workdir,
		caCertPath:              caCertPath,
		adminKeyPath:            adminKeyPath,
		adminCertPath:           adminCertPath,
		scheme:                  scheme,
		usedPorts:               usedPorts,
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
		conf.EtcdBinary,
		conf.KubeApiserverBinary,
		conf.KubeControllerManagerBinary,
		conf.KubeSchedulerBinary,
		conf.KwokControllerBinary,
		conf.PrometheusBinary,
		conf.MetricsServerBinary,
		conf.KubectlBinary,
	}, nil
}

// ListImages list images in the cluster
func (c *Cluster) ListImages(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
