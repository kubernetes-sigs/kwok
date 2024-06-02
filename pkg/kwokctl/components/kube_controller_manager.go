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
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKubeControllerManagerComponentConfig is the configuration for building a kube-controller-manager component.
type BuildKubeControllerManagerComponentConfig struct {
	Runtime                            string
	ProjectName                        string
	Binary                             string
	Image                              string
	Version                            version.Version
	Workdir                            string
	BindAddress                        string
	Port                               uint32
	SecurePort                         bool
	CaCertPath                         string
	AdminCertPath                      string
	AdminKeyPath                       string
	KubeControllerManagerCertPath      string // Add field for kube-controller-manager specific cert
	KubeControllerManagerKeyPath       string 
	KubeAuthorization                  bool
	KubeconfigPath                     string
	KubeFeatureGates                   string
	NodeMonitorPeriodMilliseconds      int64
	NodeMonitorGracePeriodMilliseconds int64
	Verbosity                          log.Level
	DisableQPSLimits                   bool
}

// BuildKubeControllerManagerComponent builds a kube-controller-manager component.
func BuildKubeControllerManagerComponent(conf BuildKubeControllerManagerComponentConfig) (component internalversion.Component, err error) {
	kubeControllerManagerArgs := []string{}

	if conf.KubeFeatureGates != "" {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--feature-gates="+conf.KubeFeatureGates,
		)
	}

	if conf.NodeMonitorPeriodMilliseconds > 0 {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--node-monitor-period="+format.String(time.Duration(conf.NodeMonitorPeriodMilliseconds)*time.Millisecond),
		)
	}

	if conf.NodeMonitorGracePeriodMilliseconds > 0 {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--node-monitor-grace-period="+format.String(time.Duration(conf.NodeMonitorGracePeriodMilliseconds)*time.Millisecond),
		)
	}

	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric

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
				HostPath:  conf.KubeControllerManagerCertPath,
				MountPath: "/etc/kubernetes/pki/kube-controller-manager.crt",
				ReadOnly:  true,
			},
			internalversion.Volume{
				HostPath:  conf.KubeControllerManagerKeyPath,
				MountPath: "/etc/kubernetes/pki/kube-controller-manager.key",
				ReadOnly:  true,
			},
		)
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--kubeconfig=/root/.kube/config",
		)
	} else {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--kubeconfig="+conf.KubeconfigPath,
		)
	}

	if conf.SecurePort {
		if conf.Version.GE(version.NewVersion(1, 13, 0)) {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics",
			)
		}

		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port=10257",
				"--tls-cert-file=/etc/kubernetes/pki/kube-controller-manager.crt", // Add argument for kube-controller-manager specific cert
				"--tls-private-key-file=/etc/kubernetes/pki/kube-controller-manager.key", // Add argument for kube-controller-manager specific key
			)
			if conf.Port > 0 {
				ports = append(
					ports,
					internalversion.Port{
						HostPort: conf.Port,
						Port:     10257,
					},
				)
			}
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               conf.ProjectName + "-" + consts.ComponentKubeControllerManager + ":10257",
				Path:               "/metrics",
				CertPath:           "/etc/kubernetes/pki/kube-controller-manager.crt", // Update metric to use kube-controller-manager specific cert
				KeyPath:            "/etc/kubernetes/pki/kube-controller-manager.key", // Update metric to use kube-controller-manager specific key
				InsecureSkipVerify: true,
			}
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port="+format.String(conf.Port),
				"--tls-cert-file="+conf.KubeControllerManagerCertPath, // Add argument for kube-controller-manager specific cert
				"--tls-private-key-file="+conf.KubeControllerManagerKeyPath, // Add argument for kube-controller-manager specific key
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               net.LocalAddress + ":" + format.String(conf.Port),
				Path:               "/metrics",
				CertPath:           conf.KubeControllerManagerCertPath, // Update metric to use kube-controller-manager specific cert
				KeyPath:            conf.KubeControllerManagerKeyPath, // Update metric to use kube-controller-manager specific key
				InsecureSkipVerify: true,
			}
		}

		// TODO: Support disable insecure port
		//	kubeControllerManagerArgs = append(kubeControllerManagerArgs,
		//		"--port=0",
		//	)
	} else {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--address="+conf.BindAddress,
				"--port=10252",
			)
			if conf.Port > 0 {
				ports = append(
					ports,
					internalversion.Port{
						HostPort: conf.Port,
						Port:     10252,
					},
				)
			}
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   conf.ProjectName + "-" + consts.ComponentKubeControllerManager + ":10252",
				Path:   "/metrics",
			}
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--address="+conf.BindAddress,
				"--port="+format.String(conf.Port),
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   net.LocalAddress + ":" + format.String(conf.Port),
				Path:   "/metrics",
			}
		}

		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--secure-port=0",
		)
	}

	if conf.KubeAuthorization {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--root-ca-file=/etc/kubernetes/pki/ca.crt",
				"--service-account-private-key-file=/etc/kubernetes/pki/kube-controller-manager.key", // Update to use kube-controller-manager specific key
			)
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--root-ca-file="+conf.CaCertPath,
				"--service-account-private-key-file="+conf.KubeControllerManagerKeyPath, // Update to use kube-controller-manager specific key
			)
		}
	}

	if conf.DisableQPSLimits {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--kube-api-qps="+format.String(consts.DefaultUnlimitedQPS),
			"--kube-api-burst="+format.String(consts.DefaultUnlimitedBurst),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	envs := []internalversion.Env{}

	return internalversion.Component{
		Name:    consts.ComponentKubeControllerManager,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{consts.ComponentKubeControllerManager},
		Volumes: volumes,
		Args:    kubeControllerManagerArgs,
		Ports:   ports,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		Metric:  metric,
		Envs:    envs,
	}, nil
}
