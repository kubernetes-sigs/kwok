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
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// kubeSchedulerReadyzVersion is the first Kubernetes release whose kube-scheduler serves /readyz.
var kubeSchedulerReadyzVersion = version.NewVersion(1, 31, 0)

// BuildKubeSchedulerComponentConfig is the configuration for building a kube-scheduler component.
type BuildKubeSchedulerComponentConfig struct {
	Runtime          string
	ProjectName      string
	Binary           string
	Image            string
	Version          version.Version
	Workdir          string
	BindAddress      string
	Port             uint32
	SecurePort       bool
	CaCertPath       string
	AdminCertPath    string
	AdminKeyPath     string
	ConfigPath       string
	KubeconfigPath   string
	KubeFeatureGates string
	Verbosity        log.Level
	DisableQPSLimits bool
}

// BuildKubeSchedulerComponent builds a kube-scheduler component.
func BuildKubeSchedulerComponent(conf BuildKubeSchedulerComponentConfig) (component internalversion.Component, err error) {
	var args []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric
	var probePort uint32

	if conf.KubeFeatureGates != "" {
		args = append(args,
			"--feature-gates="+conf.KubeFeatureGates,
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
		)

		if conf.ConfigPath != "" {
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.ConfigPath,
					MountPath: "/etc/kubernetes/scheduler.yaml",
					ReadOnly:  true,
				},
			)
			args = append(args,
				"--config=/etc/kubernetes/scheduler.yaml",
			)
		} else {
			args = append(args,
				"--kubeconfig="+kubeconfigPath,
			)
		}
	} else {
		if conf.ConfigPath != "" {
			args = append(args,
				"--config="+conf.ConfigPath,
			)
		} else {
			args = append(args,
				"--kubeconfig="+conf.KubeconfigPath,
			)
		}
	}

	if conf.SecurePort {
		args = append(args,
			"--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics",
		)

		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			args = append(args,
				"--bind-address="+conf.BindAddress,
				"--secure-port=10259",
			)
			ports = append(
				ports,
				internalversion.Port{
					Name:     schemeHTTPS,
					HostPort: conf.Port,
					Port:     10259,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             schemeHTTPS,
				Host:               conf.ProjectName + "-" + consts.ComponentKubeScheduler + ":10259",
				Path:               metricsPath,
				CertPath:           pkiAdminCertPath,
				KeyPath:            pkiAdminKeyPath,
				InsecureSkipVerify: true,
			}
			probePort = 10259
		} else {
			args = append(args,
				"--bind-address="+conf.BindAddress,
				"--secure-port="+format.String(conf.Port),
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
				Host:               utilsnet.LocalAddress + ":" + format.String(conf.Port),
				Path:               metricsPath,
				CertPath:           conf.AdminCertPath,
				KeyPath:            conf.AdminKeyPath,
				InsecureSkipVerify: true,
			}
			probePort = conf.Port
		}
		// TODO: Support disable insecure port
		//	args = append(args,
		//		"--port=0",
		//	)
	} else {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			args = append(args,
				"--address="+conf.BindAddress,
				"--port=10251",
			)
			ports = append(
				ports,
				internalversion.Port{
					Name:     "http",
					HostPort: conf.Port,
					Port:     10251,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   conf.ProjectName + "-" + consts.ComponentKubeScheduler + ":10251",
				Path:   metricsPath,
			}
			probePort = 10251
		} else {
			args = append(args,
				"--address="+conf.BindAddress,
				"--port="+format.String(conf.Port),
			)
			ports = append(
				ports,
				internalversion.Port{
					Name:     "http",
					HostPort: 0,
					Port:     conf.Port,
					Protocol: internalversion.ProtocolTCP,
				},
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   utilsnet.LocalAddress + ":" + format.String(conf.Port),
				Path:   metricsPath,
			}
			probePort = conf.Port
		}
		// The insecure serving flags are ignored when a configuration file is given,
		// which sets the address in healthzBindAddress instead.
		if conf.ConfigPath != "" {
			probePort = 0
		}

		// TODO: Support disable secure port
		//	args = append(args,
		//		"--secure-port=0",
		//	)
	}

	if conf.DisableQPSLimits {
		args = append(args,
			"--kube-api-qps="+format.String(consts.DefaultUnlimitedQPS),
			"--kube-api-burst="+format.String(consts.DefaultUnlimitedBurst),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	return internalversion.Component{
		Name:    consts.ComponentKubeScheduler,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command:        []string{consts.ComponentKubeScheduler},
		Volumes:        volumes,
		Args:           args,
		Binary:         conf.Binary,
		Image:          conf.Image,
		Ports:          ports,
		WorkDir:        conf.Workdir,
		Metric:         metric,
		ReadinessProbe: kubeSchedulerReadinessProbe(conf, args, probePort),
	}, nil
}

// kubeSchedulerReadinessProbe returns the readiness probe of kube-scheduler serving on port,
// or nil if it cannot be probed.
func kubeSchedulerReadinessProbe(conf BuildKubeSchedulerComponentConfig, args []string, port uint32) *internalversion.ComponentProbe {
	if port == 0 {
		return nil
	}
	// /readyz, which also waits for the event handlers to sync, was added in Kubernetes 1.31.
	// The scheduler-plugins image reports its own version, which falls back to /healthz, also served by it.
	path := "/healthz"
	if conf.Version.GTE(kubeSchedulerReadyzVersion) {
		path = "/readyz"
	}
	return controlPlaneReadinessProbe(conf.SecurePort, args, path, port)
}
