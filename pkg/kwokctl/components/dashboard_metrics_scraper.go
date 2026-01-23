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
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildDashboardMetricsScraperComponentConfig is the configuration for building the dashboard component.
type BuildDashboardMetricsScraperComponentConfig struct {
	Runtime string
	Binary  string
	Image   string
	Version version.Version
	Workdir string

	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
	// InCluster holds configuration for in-cluster Kubernetes client configuration.
	InCluster *InClusterConfig
}

// BuildDashboardMetricsScraperComponent builds the dashboard component.
func BuildDashboardMetricsScraperComponent(conf BuildDashboardMetricsScraperComponentConfig) (component internalversion.Component, err error) {
	dashboardArgs := []string{
		"--db-file=/metrics.db",
	}

	user := ""
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		dashboardArgs = append(dashboardArgs,
			"--kubeconfig=/root/.kube/config",
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
		user = "root"
	} else {
		dashboardArgs = append(dashboardArgs,
			"--kubeconfig="+conf.KubeconfigPath,
		)
	}

	var envs []internalversion.Env
	if conf.InCluster != nil {
		volumes = append(volumes, InClusterVolumes(*conf.InCluster)...)
		envs = append(envs, InClusterEnvs(*conf.InCluster)...)
	}

	component = internalversion.Component{
		Name:  consts.ComponentDashboardMetricsScraper,
		Image: conf.Image,
		Links: []string{
			consts.ComponentMetricsServer,
		},
		WorkDir: conf.Workdir,
		Ports:   ports,
		Volumes: volumes,
		Args:    dashboardArgs,
		User:    user,
		Envs:    envs,
	}
	return component, nil
}
