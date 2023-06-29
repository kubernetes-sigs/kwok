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
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildKwokControllerComponentConfig is the configuration for building a kwok controller component.
type BuildKwokControllerComponentConfig struct {
	Binary                   string
	Image                    string
	Version                  version.Version
	Workdir                  string
	BindAddress              string
	Port                     uint32
	ConfigPath               string
	KubeconfigPath           string
	CaCertPath               string
	AdminCertPath            string
	AdminKeyPath             string
	NodeName                 string
	Verbosity                log.Level
	NodeLeaseDurationSeconds uint
	ExtraArgs                []internalversion.ExtraArgs
	ExtraVolumes             []internalversion.Volume
}

// BuildKwokControllerComponent builds a kwok controller component.
func BuildKwokControllerComponent(conf BuildKwokControllerComponentConfig) (component internalversion.Component) {
	exposePort := true
	if conf.Port == 0 {
		conf.Port = 10247
		exposePort = false
	}

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
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/root/.kwok/kwok.yaml",
				ReadOnly:  true,
			},
		)

		if exposePort {
			ports = append(ports,
				internalversion.Port{
					HostPort: conf.Port,
					Port:     conf.Port,
				},
			)
		}
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig=/root/.kube/config",
			"--config=/root/.kwok/kwok.yaml",
			"--tls-cert-file=/etc/kubernetes/pki/admin.crt",
			"--tls-private-key-file=/etc/kubernetes/pki/admin.key",
			"--node-name="+conf.NodeName,
			"--node-lease-duration-seconds="+format.String(conf.NodeLeaseDurationSeconds),
		)
	} else {
		kwokControllerArgs = append(kwokControllerArgs,
			"--kubeconfig="+conf.KubeconfigPath,
			"--config="+conf.ConfigPath,
			"--tls-cert-file="+conf.AdminCertPath,
			"--tls-private-key-file="+conf.AdminKeyPath,
			"--node-name=localhost",
			"--node-lease-duration-seconds="+format.String(conf.NodeLeaseDurationSeconds),
		)
	}
	kwokControllerArgs = append(kwokControllerArgs,
		"--node-port="+format.String(conf.Port),
		"--server-address="+conf.BindAddress+":"+format.String(conf.Port),
	)

	if conf.Verbosity != log.LevelInfo {
		kwokControllerArgs = append(kwokControllerArgs, "--v="+format.String(conf.Verbosity))
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
	}
}
