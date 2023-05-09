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
	"strconv"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKubeControllerManagerComponentConfig is the configuration for building a kube-controller-manager component.
type BuildKubeControllerManagerComponentConfig struct {
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
	KubeAuthorization                  bool
	KubeconfigPath                     string
	KubeFeatureGates                   string
	NodeMonitorPeriodMilliseconds      int64
	NodeMonitorGracePeriodMilliseconds int64
	Verbosity                          int
	ExtraArgs                          []internalversion.ExtraArgs
	ExtraVolumes                       []internalversion.Volume
}

// BuildKubeControllerManagerComponent builds a kube-controller-manager component.
func BuildKubeControllerManagerComponent(conf BuildKubeControllerManagerComponentConfig) (component internalversion.Component, err error) {
	kubeControllerManagerArgs := []string{}
	kubeControllerManagerArgs = append(kubeControllerManagerArgs, extraArgsToStrings(conf.ExtraArgs)...)

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

	inContainer := conf.Image != ""
	var volumes []internalversion.Volume
	volumes = append(volumes, conf.ExtraVolumes...)
	var ports []internalversion.Port

	if inContainer {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.KubeconfigPath,
				MountPath: "/root/.kube/config",
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
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--kubeconfig=/root/.kube/config",
		)
	} else {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--kubeconfig="+conf.KubeconfigPath,
		)
	}

	if conf.SecurePort {
		if conf.Version.GE(version.NewVersion(1, 12, 0)) {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics",
			)
		}

		if inContainer {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port=10257",
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
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port="+format.String(conf.Port),
			)
		}

		// TODO: Support disable insecure port
		//	kubeControllerManagerArgs = append(kubeControllerManagerArgs,
		//		"--port=0",
		//	)
	} else {
		if inContainer {
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
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--address="+conf.BindAddress,
				"--port="+format.String(conf.Port),
			)
		}

		kubeControllerManagerArgs = append(kubeControllerManagerArgs,
			"--secure-port=0",
		)
	}

	if conf.KubeAuthorization {
		if inContainer {
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.CaCertPath,
					MountPath: "/etc/kubernetes/pki/ca.crt",
					ReadOnly:  true,
				},
			)
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--root-ca-file=/etc/kubernetes/pki/ca.crt",
				"--service-account-private-key-file=/etc/kubernetes/pki/admin.key",
			)
		} else {
			kubeControllerManagerArgs = append(kubeControllerManagerArgs,
				"--root-ca-file="+conf.CaCertPath,
				"--service-account-private-key-file="+conf.AdminKeyPath,
			)
		}
	}

	if conf.Verbosity != int(log.LevelInfo) {
		kubeControllerManagerArgs = append(kubeControllerManagerArgs, "--v="+strconv.Itoa(conf.Verbosity))
	}

	return internalversion.Component{
		Name:    "kube-controller-manager",
		Version: conf.Version.String(),
		Links: []string{
			"kube-apiserver",
		},
		Command: []string{"kube-controller-manager"},
		Volumes: volumes,
		Args:    kubeControllerManagerArgs,
		Ports:   ports,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
