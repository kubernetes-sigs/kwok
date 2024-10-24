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

// BuildPrometheusComponentConfig is the configuration for building a prometheus component.
type BuildPrometheusComponentConfig struct {
	Runtime                      string
	Binary                       string
	Image                        string
	Version                      version.Version
	Workdir                      string
	BindAddress                  string
	Port                         uint32
	ConfigPath                   string
	AdminCertPath                string
	AdminKeyPath                 string
	Verbosity                    log.Level
	DisableKubeControllerManager bool
	DisableKubeScheduler         bool
}

// BuildPrometheusComponent builds a prometheus component.
func BuildPrometheusComponent(conf BuildPrometheusComponentConfig) (component internalversion.Component, err error) {
	prometheusArgs := []string{}

	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric

	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/etc/prometheus/prometheus.yaml",
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
		ports = []internalversion.Port{
			{
				HostPort: conf.Port,
				Port:     9090,
			},
		}
		prometheusArgs = append(prometheusArgs,
			"--config.file=/etc/prometheus/prometheus.yaml",
			"--web.listen-address="+conf.BindAddress+":9090",
		)
	} else {
		prometheusArgs = append(prometheusArgs,
			"--config.file="+conf.ConfigPath,
			"--web.listen-address="+conf.BindAddress+":"+format.String(conf.Port),
		)
	}

	metric = &internalversion.ComponentMetric{
		Scheme: "http",
		Host:   net.LocalAddress + ":" + format.String(conf.Port),
		Path:   "/metrics",
	}

	if conf.Verbosity != log.LevelInfo {
		prometheusArgs = append(prometheusArgs, "--log.level="+log.ToLogSeverityLevel(conf.Verbosity))
	}

	envs := []internalversion.Env{}

	links := []string{
		consts.ComponentEtcd,
		consts.ComponentKubeApiserver,
		consts.ComponentKwokController,
	}
	if !conf.DisableKubeControllerManager {
		links = append(links, consts.ComponentKubeControllerManager)
	}
	if !conf.DisableKubeScheduler {
		links = append(links, consts.ComponentKubeScheduler)
	}

	return internalversion.Component{
		Name:    consts.ComponentPrometheus,
		Version: conf.Version.String(),
		Links:   links,
		Command: []string{consts.ComponentPrometheus},
		Ports:   ports,
		Volumes: volumes,
		Args:    prometheusArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		Metric:  metric,
		Envs:    envs,
	}, nil
}
