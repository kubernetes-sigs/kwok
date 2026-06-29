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

package components

import (
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKwokControllerComponentConfig is the configuration for building a kwok controller component.
type BuildKwokControllerComponentConfig struct {
	Runtime                           string
	ProjectName                       string
	Binary                            string
	Image                             string
	Version                           version.Version
	Workdir                           string
	BindAddress                       string
	Port                              uint32
	ConfigPath                        string
	KubeconfigPath                    string
	CaCertPath                        string
	AdminCertPath                     string
	AdminKeyPath                      string
	NodeIP                            string
	NodeName                          string
	ManageNodesWithAnnotationSelector string
	Verbosity                         log.Level
	NodeLeaseDurationSeconds          uint
	EnableCRDs                        []string
	OtlpGrpcAddress                   string
}

// BuildKwokControllerComponent builds a kwok controller component.
func BuildKwokControllerComponent(conf BuildKwokControllerComponentConfig) (component internalversion.Component) {
	var args []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric
	var metricsDiscovery *internalversion.ComponentMetric

	if conf.ManageNodesWithAnnotationSelector == "" {
		args = append(args,
			"--manage-all-nodes=true",
		)
	} else {
		args = append(args,
			"--manage-all-nodes=false",
			"--manage-nodes-with-annotation-selector="+conf.ManageNodesWithAnnotationSelector,
		)
	}

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
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/root/.kwok/kwok.yaml",
				ReadOnly:  true,
			},
		)

		ports = append(
			ports,
			internalversion.Port{
				Name:     "http",
				HostPort: conf.Port,
				Port:     10247,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		args = append(args,
			"--kubeconfig="+kubeconfigPath,
			"--config=/root/.kwok/kwok.yaml",
			"--tls-cert-file="+pkiAdminCertPath,
			"--tls-private-key-file="+pkiAdminKeyPath,
			"--node-ip="+conf.NodeIP,
			"--node-name="+conf.NodeName,
			"--node-port=10247",
			"--server-address="+conf.BindAddress+":10247",
			"--node-lease-duration-seconds="+format.String(conf.NodeLeaseDurationSeconds),
		)
	} else {
		ports = append(
			ports,
			internalversion.Port{
				Name:     "http",
				HostPort: 0,
				Port:     conf.Port,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		args = append(args,
			"--kubeconfig="+conf.KubeconfigPath,
			"--config="+conf.ConfigPath,
			"--tls-cert-file="+conf.AdminCertPath,
			"--tls-private-key-file="+conf.AdminKeyPath,
			"--node-ip="+conf.NodeIP,
			"--node-name="+conf.NodeName,
			"--node-port="+format.String(conf.Port),
			"--server-address="+conf.BindAddress+":"+format.String(conf.Port),
			"--node-lease-duration-seconds="+format.String(conf.NodeLeaseDurationSeconds),
		)
	}

	var metricsHost string
	switch GetRuntimeMode(conf.Runtime) {
	case RuntimeModeNative:
		metricsHost = utilsnet.LocalAddress + ":" + format.String(conf.Port)
	case RuntimeModeContainer:
		metricsHost = conf.ProjectName + "-" + consts.ComponentKwokController + ":10247"
	case RuntimeModeCluster:
		metricsHost = utilsnet.LocalAddress + ":10247"
	}

	if metricsHost != "" {
		metric = &internalversion.ComponentMetric{
			Scheme: "http",
			Host:   metricsHost,
			Path:   metricsPath,
		}
		metricsDiscovery = &internalversion.ComponentMetric{
			Scheme: "http",
			Host:   metricsHost,
			Path:   "/discovery/prometheus",
		}
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--v="+format.String(conf.Verbosity))
	}

	if len(conf.EnableCRDs) != 0 {
		args = append(args, "--enable-crds="+strings.Join(conf.EnableCRDs, ","))
	}

	if conf.OtlpGrpcAddress != "" {
		args = append(args,
			"--tracing-endpoint="+conf.OtlpGrpcAddress,
			"--tracing-sampling-rate-per-million=1000000",
		)
	}

	return internalversion.Component{
		Name:    consts.ComponentKwokController,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Ports:            ports,
		Command:          []string{"kwok"},
		Volumes:          volumes,
		Args:             args,
		Binary:           conf.Binary,
		Image:            conf.Image,
		Metric:           metric,
		MetricsDiscovery: metricsDiscovery,
		WorkDir:          conf.Workdir,
	}
}
