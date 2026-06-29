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
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// BuildKubectlProxyComponentConfig is the configuration for building a kubectl proxy component.
type BuildKubectlProxyComponentConfig struct {
	Runtime        string
	ProjectName    string
	Binary         string
	Image          string
	Workdir        string
	BindAddress    string
	Port           uint32
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	ConfigPath     string
	KubeconfigPath string
	Verbosity      log.Level
}

// BuildKubectlProxyComponent builds a kubectl proxy component.
func BuildKubectlProxyComponent(conf BuildKubectlProxyComponentConfig) (component internalversion.Component, err error) {
	var args []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port

	args = append(args,
		"proxy",
		"--accept-hosts=^*$",
		"--address="+conf.BindAddress,
	)

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
		args = append(args,
			"--kubeconfig="+kubeconfigPath,
			"--port=8001",
		)
		ports = append(
			ports,
			internalversion.Port{
				Name:     "http",
				HostPort: conf.Port,
				Port:     8001,
				Protocol: internalversion.ProtocolTCP,
			},
		)
	} else {
		args = append(args,
			"--kubeconfig="+conf.KubeconfigPath,
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
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	return internalversion.Component{
		Name: consts.ComponentKubeApiserverInsecureProxy,
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"kubectl"},
		Volumes: volumes,
		Args:    args,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Ports:   ports,
		WorkDir: conf.Workdir,
	}, nil
}
