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
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildJobSetComponentConfig is the configuration for building a jobset component.
type BuildJobSetComponentConfig struct {
	Runtime        string
	ProjectName    string
	Binary         string
	Image          string
	Version        version.Version
	Workdir        string
	BindAddress    string
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	ConfigPath     string
	KubeconfigPath string
	Verbosity      log.Level
}

// BuildJobSetComponent builds a jobset component.
func BuildJobSetComponent(conf BuildJobSetComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("jobset only supports container runtime for now")
	}

	var jobsetArgs []string
	var volumes []internalversion.Volume

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
			HostPath:  conf.CaCertPath,
			MountPath: "/tmp/k8s-webhook-server/serving-certs/ca.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminCertPath,
			MountPath: "/tmp/k8s-webhook-server/serving-certs/tls.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminKeyPath,
			MountPath: "/tmp/k8s-webhook-server/serving-certs/tls.key",
			ReadOnly:  true,
		},
	)

	if conf.ConfigPath != "" {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/controller_manager_config.yaml",
				ReadOnly:  true,
			},
		)
		jobsetArgs = append(jobsetArgs,
			"--config=/controller_manager_config.yaml",
		)
	}

	jobsetArgs = append(jobsetArgs,
		"--kubeconfig=/root/.kube/config",
	)
	user := "root"

	if conf.Verbosity != log.LevelInfo {
		jobsetArgs = append(jobsetArgs, "--zap-log-level="+log.ToZapLevel(conf.Verbosity))
	}

	return internalversion.Component{
		Name:    consts.ComponentJobSet,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"/manager"},
		Volumes: volumes,
		Args:    jobsetArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		User:    user,
	}, nil
}
