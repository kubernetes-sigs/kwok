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
	"context"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKwokControllerComponentConfig is the configuration for building a kwok controller component.
type BuildKwokControllerComponentConfig struct {
	Binary         string
	Image          string
	Version        version.Version
	Workdir        string
	Port           uint32
	ConfigPath     string
	KubeconfigPath string
	AdminCertPath  string
	AdminKeyPath   string
	NodeName       string
	ExtraArgs      []internalversion.ExtraArgs
	ExtraVolumes   []internalversion.Volume
}

// BuildKwokControllerComponent builds a kwok controller component.
func BuildKwokControllerComponent(conf BuildKwokControllerComponentConfig) (component internalversion.Component, err error) {
	kwokControllerArgs := []string{
		"--manage-all-nodes=true",
	}
	kwokControllerArgs = append(kwokControllerArgs, extraArgsToStrings(conf.ExtraArgs)...)

	inContainer := conf.Image != ""
	var volumes []internalversion.Volume
	volumes = append(volumes, conf.ExtraVolumes...)
	var ports []internalversion.Port

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

		objs, err := config.Load(context.Background(), conf.ConfigPath)
		if err != nil {
			return internalversion.Component{}, fmt.Errorf("failed to load config: %w", err)
		}

		// Mount log dirs
		mountDirs := make(map[string]bool)
		for _, obj := range objs {
			switch obj := obj.(type) {
			case *internalversion.ClusterLogs:
				for _, log := range obj.Spec.Logs {
					mountDirs[filepath.Dir(log.LogsFile)] = true
				}
			case *internalversion.Logs:
				for _, log := range obj.Spec.Logs {
					mountDirs[filepath.Dir(log.LogsFile)] = true
				}
			}
		}
		for dir := range mountDirs {
			volumes = append(volumes, internalversion.Volume{
				HostPath:  dir,
				MountPath: dir,
				ReadOnly:  true,
			})
		}

		if conf.Port != 0 {
			ports = append(ports,
				internalversion.Port{
					HostPort: conf.Port,
					Port:     10247,
				},
			)
		}
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig=/root/.kube/config",
			"--config=/root/.kwok/kwok.yaml",
			"--tls-cert-file=/etc/kubernetes/pki/admin.crt",
			"--tls-private-key-file=/etc/kubernetes/pki/admin.key",
			"--node-name="+conf.NodeName,
			"--node-port=10247",
		)
	} else {
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--config="+conf.ConfigPath,
			"--tls-cert-file="+conf.AdminCertPath,
			"--tls-private-key-file="+conf.AdminKeyPath,
			"--node-name=localhost",
			"--node-port="+format.String(conf.Port),
		)
	}

	return internalversion.Component{
		Name:    "kwok-controller",
		Version: conf.Version.String(),
		Links: []string{
			"kube-apiserver",
		},
		Ports:   ports,
		Command: []string{"kwok"},
		Volumes: volumes,
		Args:    kwokControllerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
