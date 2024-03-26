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
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"

	_ "embed"
)

//go:embed kind.yaml.tpl
var kindYamlTpl string

var kindYamlTemplate = template.Must(template.New("kind_config").Parse(kindYamlTpl))

// BuildKind builds the kind yaml content.
func BuildKind(conf BuildKindConfig) (string, error) {
	buf := bytes.NewBuffer(nil)

	var err error
	conf, err = expendExtrasForBuildKind(conf)
	if err != nil {
		return "", fmt.Errorf("failed to expand extras for build kind: %w", err)
	}

	conf, err = expandHostVolumePaths(conf)
	if err != nil {
		return "", fmt.Errorf("failed to expand host volume paths: %w", err)
	}

	err = kindYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kind yaml template: %w", err)
	}
	return buf.String(), nil
}

func expendExtrasForBuildKind(conf BuildKindConfig) (BuildKindConfig, error) {
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

	if conf.TracingConfigPath != "" {
		conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
			internalversion.ExtraArgs{
				Key:   "tracing-config-file",
				Value: "/etc/kubernetes/apiserver-tracing-config.yaml",
			},
		)
		conf.ApiserverExtraVolumes = append(conf.ApiserverExtraVolumes,
			internalversion.Volume{
				Name:      "apiserver-tracing-config",
				HostPath:  conf.TracingConfigPath,
				MountPath: "/etc/kubernetes/apiserver-tracing-config.yaml",
				ReadOnly:  true,
				PathType:  internalversion.HostPathFile,
			},
		)
		if conf.KubeVersion.LT(version.NewVersion(1, 22, 0)) {
			return conf, fmt.Errorf("the kube-apiserver version is less than 1.22.0, so the --jaeger-port cannot be enabled")
		} else if conf.KubeVersion.LT(version.NewVersion(1, 27, 0)) {
			conf.FeatureGates = append(conf.FeatureGates, "APIServerTracing: true")
		}
	}

	if conf.Verbosity != log.LevelInfo {
		v := format.String(log.ToKlogLevel(conf.Verbosity))
		sl := log.ToLogSeverityLevel(conf.Verbosity)
		conf.EtcdExtraArgs = append(conf.EtcdExtraArgs,
			internalversion.ExtraArgs{
				Key:   "log-level",
				Value: sl,
			},
		)
		conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
			internalversion.ExtraArgs{
				Key:   "v",
				Value: v,
			},
		)
		conf.ControllerManagerExtraArgs = append(conf.ControllerManagerExtraArgs,
			internalversion.ExtraArgs{
				Key:   "v",
				Value: v,
			},
		)
		conf.SchedulerExtraArgs = append(conf.SchedulerExtraArgs,
			internalversion.ExtraArgs{
				Key:   "v",
				Value: v,
			},
		)
	}

	if conf.DisableQPSLimits {
		conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
			internalversion.ExtraArgs{
				Key:   "max-requests-inflight",
				Value: "0",
			},
			internalversion.ExtraArgs{
				Key:   "max-mutating-requests-inflight",
				Value: "0",
			},
		)

		// FeatureGate APIPriorityAndFairness is not available before 1.17.0
		if conf.KubeVersion.GE(version.NewVersion(1, 18, 0)) {
			conf.ApiserverExtraArgs = append(conf.ApiserverExtraArgs,
				internalversion.ExtraArgs{
					Key:   "enable-priority-and-fairness",
					Value: "false",
				},
			)
		}
		conf.ControllerManagerExtraArgs = append(conf.ControllerManagerExtraArgs,
			internalversion.ExtraArgs{
				Key:   "kube-api-qps",
				Value: format.String(consts.DefaultUnlimitedQPS),
			},
			internalversion.ExtraArgs{
				Key:   "kube-api-burst",
				Value: format.String(consts.DefaultUnlimitedBurst),
			},
		)
		conf.SchedulerExtraArgs = append(conf.SchedulerExtraArgs,
			internalversion.ExtraArgs{
				Key:   "kube-api-qps",
				Value: format.String(consts.DefaultUnlimitedQPS),
			},
			internalversion.ExtraArgs{
				Key:   "kube-api-burst",
				Value: format.String(consts.DefaultUnlimitedBurst),
			},
		)
	}
	return conf, nil
}

func expandHostVolumePaths(conf BuildKindConfig) (BuildKindConfig, error) {
	var err error
	conf.EtcdExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.EtcdExtraVolumes)
	if err != nil {
		return conf, err
	}

	conf.ApiserverExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.ApiserverExtraVolumes)
	if err != nil {
		return conf, err
	}

	conf.SchedulerExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.SchedulerExtraVolumes)
	if err != nil {
		return conf, err
	}

	conf.EtcdExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.EtcdExtraVolumes)
	if err != nil {
		return conf, err
	}

	conf.ControllerManagerExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.ControllerManagerExtraVolumes)
	if err != nil {
		return conf, err
	}

	conf.KwokControllerExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.KwokControllerExtraVolumes)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

// BuildKindConfig is the configuration for building the kind config
type BuildKindConfig struct {
	KubeApiserverPort         uint32
	KubeApiserverInsecurePort uint32
	EtcdPort                  uint32
	DashboardPort             uint32
	PrometheusPort            uint32
	JaegerPort                uint32
	KwokControllerPort        uint32

	RuntimeConfig []string
	FeatureGates  []string

	AuditPolicy string
	AuditLog    string

	KubeconfigPath    string
	SchedulerConfig   string
	TracingConfigPath string
	Workdir           string

	EtcdExtraArgs                 []internalversion.ExtraArgs
	EtcdExtraVolumes              []internalversion.Volume
	ApiserverExtraArgs            []internalversion.ExtraArgs
	ApiserverExtraVolumes         []internalversion.Volume
	SchedulerExtraArgs            []internalversion.ExtraArgs
	SchedulerExtraVolumes         []internalversion.Volume
	ControllerManagerExtraArgs    []internalversion.ExtraArgs
	ControllerManagerExtraVolumes []internalversion.Volume
	Verbosity                     log.Level
	KwokControllerExtraVolumes    []internalversion.Volume
	PrometheusExtraVolumes        []internalversion.Volume

	BindAddress      string
	DisableQPSLimits bool
	KubeVersion      version.Version
}
