/*
Copyright 2025 The Kubernetes Authors.

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
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildDashboardApiComponentConfig is the configuration for building the dashboard api component.
type BuildDashboardApiComponentConfig struct {
	Runtime     string
	ProjectName string
	Binary      string
	Image       string
	Version     version.Version
	Workdir     string
	BindAddress string

	EnableMetrics bool

	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
}

// BuildDashboardApiComponent builds the dashboard api component.
func BuildDashboardApiComponent(conf BuildDashboardApiComponentConfig) (component internalversion.Component, err error) {
	dashboardApiArgs := []string{
		"--act-as-proxy",
		"--insecure-bind-address=" + conf.BindAddress,
		"--bind-address=127.0.0.1",
		"--port=8001",
		"--apiserver-skip-tls-verify",
		"--disable-csrf-protection",
	}

	if conf.EnableMetrics {
		switch GetRuntimeMode(conf.Runtime) {
		case RuntimeModeContainer:
			dashboardApiArgs = append(dashboardApiArgs, "--sidecar-host="+conf.ProjectName+"-"+consts.ComponentDashboardMetricsScraper+":8000")
		default:
			dashboardApiArgs = append(dashboardApiArgs, "--sidecar-host=127.0.0.1:8000")
		}
	} else {
		dashboardApiArgs = append(dashboardApiArgs, "--metrics-provider=none")
	}

	user := ""
	var volumes []internalversion.Volume
	//var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		dashboardApiArgs = append(dashboardApiArgs,
			"--kubeconfig=/root/.kube/config",
			"--insecure-port=8080",
		)
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
		// ports = append(ports,
		// 	internalversion.Port{
		// 		Name:     "http",
		// 		Port:     8080,
		// 		HostPort: conf.Port,
		// 		Protocol: internalversion.ProtocolTCP,
		// 	},
		// )
	} else {
		dashboardApiArgs = append(dashboardApiArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--insecure-port=8001",
		)
		// ports = append(ports,
		// 	internalversion.Port{
		// 		Name:     "http",
		// 		Port:     conf.Port,
		// 		HostPort: 0,
		// 		Protocol: internalversion.ProtocolTCP,
		// 	},
		// )
	}

	component = internalversion.Component{
		Name:    consts.ComponentDashboardApi,
		Image:   conf.Image,
		Links:   []string{consts.ComponentKubeApiserver},
		WorkDir: conf.Workdir,
		// Ports:   ports,
		Volumes: volumes,
		Command: []string{"/" + consts.ComponentDashboardApi},
		Args:    dashboardApiArgs,
		User:    user,
	}
	return component, nil
}
