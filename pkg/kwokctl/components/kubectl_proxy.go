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
	kubectlProxyArgs := []string{}

	var volumes []internalversion.Volume
	var ports []internalversion.Port

	kubectlProxyArgs = append(kubectlProxyArgs,
		"proxy",
		"--accept-hosts=^*$",
		"--address="+conf.BindAddress,
	)

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
		kubectlProxyArgs = append(kubectlProxyArgs,
			"--kubeconfig=/root/.kube/config",
			"--port=8001",
		)
		ports = []internalversion.Port{
			{
				HostPort: conf.Port,
				Port:     8001,
			},
		}
	} else {
		kubectlProxyArgs = append(kubectlProxyArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--port="+format.String(conf.Port),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		kubectlProxyArgs = append(kubectlProxyArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	envs := []internalversion.Env{}

	return internalversion.Component{
		Name: consts.ComponentKubeApiserverInsecureProxy,
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"kubectl"},
		Volumes: volumes,
		Args:    kubectlProxyArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Ports:   ports,
		WorkDir: conf.Workdir,
		Envs:    envs,
	}, nil
}
