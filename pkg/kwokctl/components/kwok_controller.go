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
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

type BuildKwokControllerComponentConfig struct {
	Binary         string
	Image          string
	Version        version.Version
	Workdir        string
	Address        string
	Port           uint32
	ConfigPath     string
	KubeconfigPath string
	AdminCertPath  string
	AdminKeyPath   string
}

func BuildKwokControllerComponent(conf BuildKwokControllerComponentConfig) (component internalversion.Component, err error) {
	if conf.Address == "" {
		conf.Address = publicAddress
	}

	kwokControllerArgs := []string{
		"--manage-all-nodes=true",
	}

	inContainer := conf.Image != ""
	var volumes []internalversion.Volume

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
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/root/.kwok/kwok.yaml",
				ReadOnly:  true,
			},
		)
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig=/root/.kube/config",
			"--config=/root/.kwok/kwok.yaml",
			"--server-address="+publicAddress+":8080",
		)
	} else {
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--config="+conf.ConfigPath,
			"--server-address="+conf.Address+":"+format.String(conf.Port),
		)
	}

	return internalversion.Component{
		Name:    "kwok-controller",
		Version: conf.Version.String(),
		Links: []string{
			"kube-apiserver",
		},
		Command: []string{"kwok"},
		Volumes: volumes,
		Args:    kwokControllerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
