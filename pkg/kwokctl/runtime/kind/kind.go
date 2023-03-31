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

package kind

import (
	"bytes"
	"fmt"
	"text/template"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"

	_ "embed"
)

//go:embed kind.yaml.tpl
var kindYamlTpl string

var kindYamlTemplate = template.Must(template.New("_").Parse(kindYamlTpl))

// BuildKind builds the kind yaml content.
func BuildKind(conf BuildKindConfig) (string, error) {
	buf := bytes.NewBuffer(nil)

	conf = expendExtrasForBuildKind(conf)

	err := kindYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kind yaml template: %w", err)
	}
	return buf.String(), nil
}

func expendExtrasForBuildKind(conf BuildKindConfig) BuildKindConfig {
	if conf.AuditPolicy != "" {
		conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
			internalversion.ExtraArgs{
				Key:   "audit-policy-file",
				Value: "/etc/kubernetes/audit/audit.yaml",
			},
		)
		conf.ApiserverExtraVolumes = append(conf.ApiserverExtraVolumes,
			internalversion.Volume{
				Name:      "audit-policy-file",
				HostPath:  conf.AuditPolicy,
				MountPath: "/etc/kubernetes/audit/audit.yaml",
				ReadOnly:  true,
				PathType:  internalversion.HostPathFile,
			},
		)

		if conf.AuditLog != "" {
			conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
				internalversion.ExtraArgs{
					Key:   "audit-log-path",
					Value: "/var/log/kubernetes/audit.log",
				},
			)
			conf.ApiserverExtraVolumes = append(conf.ApiserverExtraVolumes,
				internalversion.Volume{
					Name:      "audit-log-path",
					HostPath:  conf.AuditLog,
					MountPath: "/var/log/kubernetes/audit.log",
					ReadOnly:  false,
					PathType:  internalversion.HostPathFile,
				},
			)
		}
	}

	if conf.SchedulerConfig != "" {
		conf.SchedulerExtraArgs = append(conf.SchedulerExtraArgs,
			internalversion.ExtraArgs{
				Key:   "config",
				Value: "/etc/kubernetes/scheduler/scheduler.yaml",
			},
		)

		conf.SchedulerExtraVolumes = append(conf.SchedulerExtraVolumes,
			internalversion.Volume{
				Name:      "config",
				HostPath:  conf.SchedulerConfig,
				MountPath: "/etc/kubernetes/scheduler/scheduler.yaml",
				ReadOnly:  true,
				PathType:  internalversion.HostPathFile,
			},
		)
	}
	return conf
}

// BuildKindConfig is the configuration for building the kind config
type BuildKindConfig struct {
	KubeApiserverPort  uint32
	EtcdPort           uint32
	PrometheusPort     uint32
	KwokControllerPort uint32

	RuntimeConfig []string
	FeatureGates  []string

	AuditPolicy string
	AuditLog    string

	KubeconfigPath  string
	SchedulerConfig string
	ConfigPath      string

	EtcdExtraArgs                 []internalversion.ExtraArgs
	EtcdExtraVolumes              []internalversion.Volume
	ApiserverExtraArgs            []internalversion.ExtraArgs
	ApiserverExtraVolumes         []internalversion.Volume
	SchedulerExtraArgs            []internalversion.ExtraArgs
	SchedulerExtraVolumes         []internalversion.Volume
	ControllerManagerExtraArgs    []internalversion.ExtraArgs
	ControllerManagerExtraVolumes []internalversion.Volume
}
