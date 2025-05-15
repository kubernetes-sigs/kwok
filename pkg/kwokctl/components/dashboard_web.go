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
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildDashboardWebComponentConfig is the configuration for building the dashboard web component.
type BuildDashboardWebComponentConfig struct {
	Runtime     string
	ProjectName string
	Binary      string
	Image       string
	Version     version.Version
	Workdir     string
	BindAddress string
	Port        uint32

	Banner string

	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
}

// BuildDashboardWebComponent builds the dashboard web component.
func BuildDashboardWebComponent(conf BuildDashboardWebComponentConfig) (component internalversion.Component, err error) {
	dashboardWebArgs := []string{
		"--bind-address=127.0.0.1",
		"--port=0",
		"--system-banner=" + conf.Banner,
		//	"--auto-generate-certificates",
	}

	user := ""
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		dashboardWebArgs = append(dashboardWebArgs,
			"--kubeconfig=/root/.kube/config",
			"--insecure-bind-address="+conf.BindAddress,
			"--insecure-port=8000",
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
		ports = append(ports,
			internalversion.Port{
				Name:     "http",
				Port:     8000,
				HostPort: conf.Port,
				Protocol: internalversion.ProtocolTCP,
			},
		)
	} else {
		dashboardWebArgs = append(dashboardWebArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--insecure-bind-address="+conf.BindAddress,
			"--insecure-port="+format.String(conf.Port),
		)
		ports = append(ports,
			internalversion.Port{
				Name:     "http",
				Port:     conf.Port,
				HostPort: 0,
				Protocol: internalversion.ProtocolTCP,
			},
		)
	}

	component = internalversion.Component{
		Name:    consts.ComponentDashboardWeb,
		Image:   conf.Image,
		Links:   []string{consts.ComponentKubeApiserver, consts.ComponentDashboardApi},
		WorkDir: conf.Workdir,
		Ports:   ports,
		Volumes: volumes,
		Command: []string{"/" + consts.ComponentDashboardWeb},
		Args:    dashboardWebArgs,
		User:    user,
	}
	return component, nil
}
