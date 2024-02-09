/*
Copyright 2024 The Kubernetes Authors.

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

package components

import (
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildMetricsServerComponentConfig is the configuration for building a metrics server component.
type BuildMetricsServerComponentConfig struct {
	Runtime        string
	ProjectName    string
	Binary         string
	Image          string
	Version        version.Version
	Workdir        string
	BindAddress    string
	Port           uint32
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
	Verbosity      log.Level
	ExtraArgs      []string
}

// BuildMetricsServerComponent builds a metrics server component.
func BuildMetricsServerComponent(conf BuildMetricsServerComponentConfig) (component internalversion.Component, err error) {
	metricsServerArgs := []string{
		"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
		"--kubelet-use-node-status-port",
		"--kubelet-insecure-tls", // TODO: remove this flag
		"--metric-resolution=15s",
	}

	var metricsHost string
	switch GetRuntimeMode(conf.Runtime) {
	case RuntimeModeNative:
		metricsHost = net.LocalAddress + ":" + format.String(conf.Port)
	case RuntimeModeContainer:
		metricsHost = conf.ProjectName + "-" + consts.ComponentKwokController + ":4443"
	case RuntimeModeCluster:
		metricsHost = net.LocalAddress + ":4443"
	}

	var metric *internalversion.ComponentMetric

	user := ""
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.KubeconfigPath,
				MountPath: "/root/.kube/config",
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.CaCertPath,
				MountPath: "/etc/kubernetes/pki/ca.crt",
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.AdminCertPath,
				MountPath: "/etc/kubernetes/pki/admin.crt",
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.AdminKeyPath,
				MountPath: "/etc/kubernetes/pki/admin.key",
				ReadOnly:  true,
			},
		)

		metricsServerArgs = append(metricsServerArgs,
			"--bind-address="+conf.BindAddress,
			"--secure-port=4443",
			"--kubeconfig=/root/.kube/config",
			"--authentication-kubeconfig=/root/.kube/config",
			"--authorization-kubeconfig=/root/.kube/config",
			"--tls-cert-file=/etc/kubernetes/pki/admin.crt",
			"--tls-private-key-file=/etc/kubernetes/pki/admin.key",
		)
		if conf.Port != 0 {
			ports = []internalversion.Port{
				{
					HostPort: conf.Port,
					Port:     4443,
				},
			}
		}
		metric = &internalversion.ComponentMetric{
			Scheme:             "https",
			Host:               metricsHost,
			Path:               "/metrics",
			CertPath:           "/etc/kubernetes/pki/admin.crt",
			KeyPath:            "/etc/kubernetes/pki/admin.key",
			InsecureSkipVerify: true,
		}
		user = "root"
	} else {
		metricsServerArgs = append(metricsServerArgs,
			"--bind-address="+conf.BindAddress,
			"--secure-port="+format.String(conf.Port),
			"--kubeconfig="+conf.KubeconfigPath,
			"--authentication-kubeconfig="+conf.KubeconfigPath,
			"--authorization-kubeconfig="+conf.KubeconfigPath,
			"--tls-cert-file="+conf.AdminCertPath,
			"--tls-private-key-file="+conf.AdminKeyPath,
		)

		metric = &internalversion.ComponentMetric{
			Scheme:             "https",
			Host:               metricsHost,
			Path:               "/metrics",
			CertPath:           conf.AdminCertPath,
			KeyPath:            conf.AdminKeyPath,
			InsecureSkipVerify: true,
		}
	}

	if conf.Verbosity != log.LevelInfo {
		metricsServerArgs = append(metricsServerArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	metricsServerArgs = AddExtraArgs(metricsServerArgs, conf.ExtraArgs)

	envs := []internalversion.Env{}

	return internalversion.Component{
		Name:    consts.ComponentMetricsServer,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKwokController,
		},
		Command: []string{"/metrics-server"},
		User:    user,
		Ports:   ports,
		Volumes: volumes,
		Args:    metricsServerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Metric:  metric,
		WorkDir: conf.Workdir,
		Envs:    envs,
	}, nil
}
