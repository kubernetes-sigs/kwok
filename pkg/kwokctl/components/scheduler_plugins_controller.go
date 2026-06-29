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
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildSchedulerPluginsControllerComponentConfig is the configuration for building the scheduler-plugins controller component.
type BuildSchedulerPluginsControllerComponentConfig struct {
	Runtime        string
	Image          string
	RawManifests   []string
	Version        version.Version
	KubeconfigPath string
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	Workdir        string
}

// BuildSchedulerPluginsControllerComponent builds the scheduler-plugins controller component.
func BuildSchedulerPluginsControllerComponent(conf BuildSchedulerPluginsControllerComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("scheduler-plugins only supports container runtime for now")
	}

	var args []string
	var volumes []internalversion.Volume

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
	)

	component = internalversion.Component{
		Name:  consts.ComponentSchedulerPlugins,
		Image: conf.Image,
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"controller"},
		Args:    args,
		Volumes: volumes,
		Version: conf.Version.String(),
		WorkDir: conf.Workdir,
	}

	if len(conf.RawManifests) != 0 {
		for _, manifest := range conf.RawManifests {
			manifestContent, err := BuildSchedulerPluginsManifest(BuildSchedulerPluginsManifestConfig{
				RawManifest: manifest,
			})
			if err != nil {
				return internalversion.Component{}, err
			}
			component.ManifestContents = append(component.ManifestContents, manifestContent...)
		}
	} else {
		component.ManifestContents = []string{}
	}
	return component, nil
}
