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

// BuildKueueComponentConfig is the configuration for building a kueue component.
type BuildKueueComponentConfig struct {
	Runtime           string
	ProjectName       string
	Binary            string
	Image             string
	Version           version.Version
	Workdir           string
	KubeApiserverPort uint32
	BindAddress       string
	Port              uint32
	CaCertPath        string
	AdminCertPath     string
	AdminKeyPath      string
	ConfigPath        string
	KubeconfigPath    string
	Verbosity         log.Level
}

// BuildKueueComponent builds a kueue component.
func BuildKueueComponent(conf BuildKueueComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("kueue only supports container runtime for now")
	}

	var kueueArgs []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric

	kueueArgs = append(kueueArgs,
		// Remove this after https://github.com/kubernetes-sigs/kueue/issues/8606 is fixed
		"--feature-gates=VisibilityOnDemand=false",
	)

	envs := []internalversion.Env{
		{
			Name:  "NAMESPACE",
			Value: "kueue-system",
		},
	}

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
			MountPath: "/etc/kueue/metrics/certs/ca.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminCertPath,
			MountPath: "/etc/kueue/metrics/certs/tls.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminKeyPath,
			MountPath: "/etc/kueue/metrics/certs/tls.key",
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
		internalversion.Volume{
			HostPath:  conf.CaCertPath,
			MountPath: "/visibility/ca.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminCertPath,
			MountPath: "/visibility/tls.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  conf.AdminKeyPath,
			MountPath: "/visibility/tls.key",
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
		kueueArgs = append(kueueArgs,
			"--config=/controller_manager_config.yaml",
		)
	}

	kueueArgs = append(kueueArgs,
		"--kubeconfig=/root/.kube/config",
	)
	user := "root"

	metric = &internalversion.ComponentMetric{
		Scheme: "https",
		Host:   conf.ProjectName + "-" + consts.ComponentKueue + ":8443",
		Path:   "/metrics",
	}

	if conf.Verbosity != log.LevelInfo {
		kueueArgs = append(kueueArgs, "--zap-log-level="+log.ToZapLevel(conf.Verbosity))
	}

	return internalversion.Component{
		Name:    consts.ComponentKueue,
		Version: conf.Version.String(),
		Links: []string{
			consts.ComponentKubeApiserver,
		},
		Command: []string{"/manager"},
		Volumes: volumes,
		Args:    kueueArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		Ports:   ports,
		WorkDir: conf.Workdir,
		Metric:  metric,
		Envs:    envs,
		User:    user,
	}, nil
}
