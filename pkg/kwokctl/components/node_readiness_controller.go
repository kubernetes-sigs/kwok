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

// BuildNodeReadinessControllerComponentConfig is the configuration for building a node-readiness-controller component.
type BuildNodeReadinessControllerComponentConfig struct {
	Runtime        string
	ProjectName    string
	Image          string
	RawManifests   []string
	Version        version.Version
	Workdir        string
	BindAddress    string
	CaCertPath     string
	AdminCertPath  string
	AdminKeyPath   string
	KubeconfigPath string
	Verbosity      log.Level
}

// BuildNodeReadinessControllerComponent builds a node-readiness-controller component.
func BuildNodeReadinessControllerComponent(conf BuildNodeReadinessControllerComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("node-readiness-controller only supports container runtime for now")
	}

	var args []string
	var volumes []internalversion.Volume
	var metric *internalversion.ComponentMetric

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
		"--leader-elect=false",
		"--metrics-bind-address=:8080",
		"--metrics-secure=false",
	)

	metric = &internalversion.ComponentMetric{
		Scheme:             "http",
		Host:               conf.ProjectName + "-" + consts.ComponentNodeReadinessController + ":8080",
		Path:               metricsPath,
		InsecureSkipVerify: true,
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--zap-log-level="+format.String(log.ToZapLevel(conf.Verbosity)))
	}

	component = internalversion.Component{
		Name:    consts.ComponentNodeReadinessController,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"/manager"},
		Volumes: volumes,
		Args:    args,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		Metric:  metric,
	}

	if len(conf.RawManifests) != 0 {
		for _, rawManifest := range conf.RawManifests {
			manifest, err := BuildNodeReadinessControllerManifest(BuildNodeReadinessControllerManifestConfig{
				RawManifest: rawManifest,
			})
			if err != nil {
				return internalversion.Component{}, err
			}
			component.ManifestContents = append(component.ManifestContents, manifest...)
		}
	} else {
		component.ManifestContents = []string{}
	}
	return component, nil
}
