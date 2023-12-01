/*
Copyright 2023 The Kubernetes Authors.

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

// BuildDashboardComponentConfig is the configuration for building the dashboard component.
type BuildDashboardComponentConfig struct {
	Runtime     string
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

// BuildDashboardComponent builds the dashboard component.
func BuildDashboardComponent(conf BuildDashboardComponentConfig) (component internalversion.Component, err error) {
	dashboardArgs := []string{
		"--insecure-bind-address=" + conf.BindAddress,
		"--bind-address=127.0.0.1",
		"--port=0",
		"--enable-insecure-login",
		"--enable-skip-login",
		"--disable-settings-authorizer",
		"--metrics-provider=none",
	}
	if conf.Banner != "" {
		dashboardArgs = append(dashboardArgs, "--system-banner="+conf.Banner)
	}

	user := ""
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		dashboardArgs = append(dashboardArgs,
			"--kubeconfig=/root/.kube/config",
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
		dashboardArgs = append(dashboardArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--insecure-port="+format.String(conf.Port),
		)
	}

	component = internalversion.Component{
		Name:  consts.ComponentDashboard,
		Image: conf.Image,
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		WorkDir: conf.Workdir,
		Ports:   ports,
		Volumes: volumes,
		Args:    dashboardArgs,
		User:    user,
	}
	return component, nil
}
