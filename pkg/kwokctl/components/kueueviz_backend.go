/*
Copyright 2026 The Kubernetes Authors.

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
	"fmt"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
)

// BuildKueuevizBackendComponentConfig is the configuration for building a kueueviz backend component.
type BuildKueuevizBackendComponentConfig struct {
	Runtime        string
	ProjectName    string
	Binary         string
	Image          string
	Port           uint32
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
}

// BuildKueuevizBackendComponent builds a kueueviz backend component.
func BuildKueuevizBackendComponent(conf BuildKueuevizBackendComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("kueue only supports container runtime for now")
	}

	var volumes []internalversion.Volume
	var ports []internalversion.Port

	volumes = append(volumes,
		internalversion.Volume{
			HostPath:  conf.KubeconfigPath,
			MountPath: "/home/nonroot/.kube/config",
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

	ports = append(
		ports,
		internalversion.Port{
			Name:     "http",
			HostPort: conf.Port,
			Port:     8080,
			Protocol: internalversion.ProtocolTCP,
		},
	)

	return internalversion.Component{
		Name: consts.ComponentKueueviz + "-backend",
		Links: []string{
			consts.ComponentKubeApiserver,
			consts.ComponentKueue,
		},
		Volumes: volumes,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Ports:   ports,
	}, nil
}
