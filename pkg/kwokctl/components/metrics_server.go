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
	"net"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildMetricsServerComponentConfig is the configuration for building a metrics server component.
type BuildMetricsServerComponentConfig struct {
	Runtime        string
	ProjectName    string
	Binary         string
	Image          string
	RawManifests   []string
	Version        version.Version
	Workdir        string
	BindAddress    string
	Port           uint32
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
	Verbosity      log.Level
}

// BuildMetricsServerComponent builds a metrics server component.
func BuildMetricsServerComponent(conf BuildMetricsServerComponentConfig) (component internalversion.Component, err error) {
	metricsServerArgs := []string{
		"--kubelet-preferred-address-types=Hostname,InternalIP,ExternalIP",
		"--kubelet-use-node-status-port",
		"--kubelet-insecure-tls", // TODO: remove this flag
		"--metric-resolution=15s",
	}

	var metricsHost string
	var metricsPort uint32
	switch GetRuntimeMode(conf.Runtime) {
	case RuntimeModeNative:
		metricsHost = utilsnet.LocalAddress
		metricsPort = conf.Port
	case RuntimeModeContainer:
		metricsHost = conf.ProjectName + "-" + consts.ComponentMetricsServer
		metricsPort = 4443
	case RuntimeModeCluster:
		metricsHost = utilsnet.LocalAddress
		metricsPort = 4443
	}

	metricsAddress := net.JoinHostPort(metricsHost, format.String(metricsPort))

	var metric *internalversion.ComponentMetric

	user := ""
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.KubeconfigPath,
				MountPath: kubeconfigPath,
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.CaCertPath,
				MountPath: pkiCACertPath,
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.AdminCertPath,
				MountPath: pkiAdminCertPath,
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.AdminKeyPath,
				MountPath: pkiAdminKeyPath,
				ReadOnly:  true,
			},
		)

		metricsServerArgs = append(metricsServerArgs,
			"--bind-address="+conf.BindAddress,
			"--secure-port=4443",
			"--kubeconfig="+kubeconfigPath,
			"--authentication-kubeconfig="+kubeconfigPath,
			"--authorization-kubeconfig="+kubeconfigPath,
			"--tls-cert-file="+pkiAdminCertPath,
			"--tls-private-key-file="+pkiAdminKeyPath,
		)
		ports = append(
			ports,
			internalversion.Port{
				Name:     schemeHTTPS,
				HostPort: conf.Port,
				Port:     4443,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		metric = &internalversion.ComponentMetric{
			Scheme:             schemeHTTPS,
			Host:               metricsAddress,
			Path:               metricsPath,
			CertPath:           pkiAdminCertPath,
			KeyPath:            pkiAdminKeyPath,
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
		ports = append(
			ports,
			internalversion.Port{
				Name:     schemeHTTPS,
				HostPort: 0,
				Port:     conf.Port,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		metric = &internalversion.ComponentMetric{
			Scheme:             schemeHTTPS,
			Host:               metricsAddress,
			Path:               metricsPath,
			CertPath:           conf.AdminCertPath,
			KeyPath:            conf.AdminKeyPath,
			InsecureSkipVerify: true,
		}
	}

	if conf.Verbosity != log.LevelInfo {
		metricsServerArgs = append(metricsServerArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	envs := []internalversion.Env{}

	component = internalversion.Component{
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
	}

	if len(conf.RawManifests) != 0 {
		for _, manifest := range conf.RawManifests {
			manifestContents, err := BuildMetricsServerManifest(BuildMetricsServerManifestConfig{
				Port:         metricsPort,
				ExternalName: metricsHost,
				RawManifest:  manifest,
			})
			if err != nil {
				return internalversion.Component{}, err
			}
			component.ManifestContents = append(component.ManifestContents, manifestContents...)
		}
	} else {
		component.ManifestContents = []string{}
	}
	return component, nil
}
