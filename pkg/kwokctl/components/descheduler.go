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
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

const deschedulerKubeconfigPath = "/tmp/kubeconfig"

// BuildDeschedulerComponentConfig is the configuration for building a descheduler component.
type BuildDeschedulerComponentConfig struct {
	Runtime        string
	Binary         string
	Image          string
	RawManifest    string
	Version        version.Version
	Workdir        string
	BindAddress    string
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
	ConfigPath     string
	Verbosity      log.Level
}

// BuildDeschedulerComponent builds a descheduler component.
func BuildDeschedulerComponent(conf BuildDeschedulerComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("descheduler only supports container runtime for now")
	}

	var deschedulerArgs []string
	var volumes []internalversion.Volume

	volumes = append(volumes,
		internalversion.Volume{
			HostPath:  conf.KubeconfigPath,
			MountPath: deschedulerKubeconfigPath,
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

	deschedulerArgs = append(deschedulerArgs,
		"--kubeconfig="+deschedulerKubeconfigPath,
		"--descheduling-interval=5m",
	)

	if conf.ConfigPath != "" {
		volumes = append(volumes,
			internalversion.Volume{
				HostPath:  conf.ConfigPath,
				MountPath: "/policy-config-file.yaml",
				ReadOnly:  true,
			},
		)
		deschedulerArgs = append(deschedulerArgs,
			"--policy-config-file=/policy-config-file.yaml",
		)
	}

	if conf.Verbosity != log.LevelInfo {
		deschedulerArgs = append(deschedulerArgs, "--v="+format.String(log.ToKlogLevel(conf.Verbosity)))
	}

	user := "0"

	component = internalversion.Component{
		Name:    consts.ComponentDescheduler,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"/bin/descheduler"},
		Volumes: volumes,
		Args:    deschedulerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		User:    user,
	}

	if conf.RawManifest != "" {
		component.ManifestContents, err = BuildDeschedulerManifest(BuildDeschedulerManifestConfig{
			RawManifest: conf.RawManifest,
		})
		if err != nil {
			return internalversion.Component{}, err
		}
	} else {
		component.ManifestContents = []string{}
	}
	return component, nil
}
