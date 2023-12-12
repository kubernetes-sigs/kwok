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
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

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
	ExtraArgs        []internalversion.ExtraArgs
	ExtraVolumes     []internalversion.Volume
	ExtraEnvs        []internalversion.Env
}

// BuildKubeSchedulerComponent builds a kube-scheduler component.
func BuildKubeSchedulerComponent(conf BuildKubeSchedulerComponentConfig) (component internalversion.Component, err error) {
	kubeSchedulerArgs := []string{}
	kubeSchedulerArgs = append(kubeSchedulerArgs, extraArgsToStrings(conf.ExtraArgs)...)

	if conf.KubeFeatureGates != "" {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--feature-gates="+conf.KubeFeatureGates,
		)
	}

	var volumes []internalversion.Volume
	volumes = append(volumes, conf.ExtraVolumes...)
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

		if conf.ConfigPath != "" {
			volumes = append(volumes,
				internalversion.Volume{
					HostPath:  conf.ConfigPath,
					MountPath: "/etc/kubernetes/scheduler.yaml",
					ReadOnly:  true,
				},
			)
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--config=/etc/kubernetes/scheduler.yaml",
			)
		} else {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--kubeconfig=/root/.kube/config",
			)
		}
	} else {
		if conf.ConfigPath != "" {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--config="+conf.ConfigPath,
			)
		} else {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--kubeconfig="+conf.KubeconfigPath,
			)
		}
	}

	if conf.SecurePort {
		if conf.Version.GE(version.NewVersion(1, 13, 0)) {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--authorization-always-allow-paths=/healthz,/readyz,/livez,/metrics",
			)
		}

		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port=10259",
			)
			if conf.Port != 0 {
				ports = append(
					ports,
					internalversion.Port{
						HostPort: conf.Port,
						Port:     10259,
					},
				)
			}
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               conf.ProjectName + "-" + consts.ComponentKubeScheduler + ":10259",
				Path:               "/metrics",
				CertPath:           "/etc/kubernetes/pki/admin.crt",
				KeyPath:            "/etc/kubernetes/pki/admin.key",
				InsecureSkipVerify: true,
			}
		} else {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--bind-address="+conf.BindAddress,
				"--secure-port="+format.String(conf.Port),
			)
			metric = &internalversion.ComponentMetric{
				Scheme:             "https",
				Host:               net.LocalAddress + ":" + format.String(conf.Port),
				Path:               "/metrics",
				CertPath:           conf.AdminCertPath,
				KeyPath:            conf.AdminKeyPath,
				InsecureSkipVerify: true,
			}
		}
		// TODO: Support disable insecure port
		//	kubeSchedulerArgs = append(kubeSchedulerArgs,
		//		"--port=0",
		//	)
	} else {
		if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--address="+conf.BindAddress,
				"--port=10251",
			)
			if conf.Port != 0 {
				ports = append(
					ports,
					internalversion.Port{
						HostPort: conf.Port,
						Port:     10251,
					},
				)
			}
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   conf.ProjectName + "-" + consts.ComponentKubeScheduler + ":10251",
				Path:   "/metrics",
			}
		} else {
			kubeSchedulerArgs = append(kubeSchedulerArgs,
				"--address="+conf.BindAddress,
				"--port="+format.String(conf.Port),
			)
			metric = &internalversion.ComponentMetric{
				Scheme: "http",
				Host:   net.LocalAddress + ":" + format.String(conf.Port),
				Path:   "/metrics",
			}
		}

		// TODO: Support disable secure port
		//	kubeSchedulerArgs = append(kubeSchedulerArgs,
		//		"--secure-port=0",
		//	)
	}

	if conf.DisableQPSLimits {
		kubeSchedulerArgs = append(kubeSchedulerArgs,
			"--kube-api-qps="+format.String(consts.DefaultUnlimitedQPS),
			"--kube-api-burst="+format.String(consts.DefaultUnlimitedBurst),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		kubeSchedulerArgs = append(kubeSchedulerArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	envs := []internalversion.Env{}
	envs = append(envs, conf.ExtraEnvs...)

	return internalversion.Component{
		Name:    consts.ComponentKubeScheduler,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{consts.ComponentKubeScheduler},
		Volumes: volumes,
		Args:    kubeSchedulerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Ports:   ports,
		WorkDir: conf.Workdir,
		Metric:  metric,
		Envs:    envs,
	}, nil
}
