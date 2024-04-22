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
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	kindv1alpha4 "sigs.k8s.io/kwok/pkg/kwokctl/runtime/kind/config/kind/v1alpha4"
	kubeadmv1beta3 "sigs.k8s.io/kwok/pkg/kwokctl/runtime/kind/config/kubeadm/v1beta3"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// BuildKind builds the kind yaml content.
func BuildKind(conf BuildKindConfig) (string, error) {
	var err error
	conf, err = expendExtrasForBuildKind(conf)
	if err != nil {
		return "", fmt.Errorf("failed to expand extras for build kind: %w", err)
	}

	conf, err = expandHostVolumePaths(conf)
	if err != nil {
		return "", fmt.Errorf("failed to expand host volume paths: %w", err)
	}

	kubeadmConfig, err := buildKubeadmConfigV1beta3(conf)
	if err != nil {
		return "", fmt.Errorf("failed to build kubeadm config: %w", err)
	}
	kubeadmConfigData, err := yaml.Marshal(kubeadmConfig)
	if err != nil {
		return "", err
	}

	kindConfig, err := buildKindConfigV1alpha4(conf)
	if err != nil {
		return "", fmt.Errorf("failed to build kind config: %w", err)
	}
	kindConfig.KubeadmConfigPatches = []string{string(kubeadmConfigData)}
	kindConfigData, err := yaml.Marshal(kindConfig)
	if err != nil {
		return "", err
	}

	return string(kindConfigData), nil
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
			conf.FeatureGates = append(conf.FeatureGates, "APIServerTracing=true")
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
	KubeApiserverPort  uint32
	EtcdPort           uint32
	DashboardPort      uint32
	PrometheusPort     uint32
	JaegerPort         uint32
	KwokControllerPort uint32

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

func buildKindConfigV1alpha4(conf BuildKindConfig) (*kindv1alpha4.Cluster, error) {
	var extraPortMappings []kindv1alpha4.PortMapping

	if conf.DashboardPort != 0 {
		extraPortMappings = append(extraPortMappings, kindv1alpha4.PortMapping{
			ContainerPort: 8080,
			HostPort:      int32(conf.DashboardPort),
			Protocol:      kindv1alpha4.PortMappingProtocolTCP,
		})
	}

	if conf.PrometheusPort != 0 {
		extraPortMappings = append(extraPortMappings, kindv1alpha4.PortMapping{
			ContainerPort: 9090,
			HostPort:      int32(conf.PrometheusPort),
			Protocol:      kindv1alpha4.PortMappingProtocolTCP,
		})
	}

	if conf.JaegerPort != 0 {
		extraPortMappings = append(extraPortMappings, kindv1alpha4.PortMapping{
			ContainerPort: 16686,
			HostPort:      int32(conf.JaegerPort),
			Protocol:      kindv1alpha4.PortMappingProtocolTCP,
		})
	}

	if conf.KwokControllerPort != 0 {
		extraPortMappings = append(extraPortMappings, kindv1alpha4.PortMapping{
			ContainerPort: 10247,
			HostPort:      int32(conf.KwokControllerPort),
			Protocol:      kindv1alpha4.PortMappingProtocolTCP,
		})
	}

	if conf.EtcdPort != 0 {
		extraPortMappings = append(extraPortMappings, kindv1alpha4.PortMapping{
			ContainerPort: 2379,
			HostPort:      int32(conf.EtcdPort),
			Protocol:      kindv1alpha4.PortMappingProtocolTCP,
		})
	}

	var extraMounts = []kindv1alpha4.Mount{
		{
			HostPath:      conf.Workdir,
			ContainerPath: "/etc/kwok/",
		},
		{
			HostPath:      fmt.Sprintf("%s/manifests", conf.Workdir),
			ContainerPath: "/etc/kubernetes/manifests",
		},
		{
			HostPath:      fmt.Sprintf("%s/pki", conf.Workdir),
			ContainerPath: "/etc/kubernetes/pki",
		},
	}

	for _, vol := range conf.EtcdExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/etcd%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	for _, vol := range conf.ApiserverExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/apiserver%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	for _, vol := range conf.ControllerManagerExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/controller-manager%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	for _, vol := range conf.SchedulerExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/scheduler%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	for _, vol := range conf.KwokControllerExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/controller%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	for _, vol := range conf.PrometheusExtraVolumes {
		extraMounts = append(extraMounts, kindv1alpha4.Mount{
			HostPath:      vol.HostPath,
			ContainerPath: fmt.Sprintf("/var/components/prometheus%s", vol.MountPath),
			Readonly:      vol.ReadOnly,
		})
	}

	featureGates := map[string]bool{}
	for _, fg := range conf.FeatureGates {
		kv := strings.SplitN(fg, "=", 2)
		if len(kv) == 2 {
			featureGates[kv[0]], _ = strconv.ParseBool(kv[1])
		} else {
			return nil, fmt.Errorf("invalid feature gate: %s", fg)
		}
	}

	runtimeConfig := map[string]string{}
	for _, rc := range conf.RuntimeConfig {
		kv := strings.SplitN(rc, "=", 2)
		if len(kv) == 2 {
			runtimeConfig[kv[0]] = kv[1]
		} else {
			return nil, fmt.Errorf("invalid runtime config: %s", rc)
		}
	}

	c := kindv1alpha4.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "kind.x-k8s.io/v1alpha4",
		},
		FeatureGates:  featureGates,
		RuntimeConfig: runtimeConfig,

		Networking: kindv1alpha4.Networking{
			APIServerPort: int32(conf.KubeApiserverPort),
		},

		Nodes: []kindv1alpha4.Node{
			{
				Role:              kindv1alpha4.ControlPlaneRole,
				ExtraPortMappings: extraPortMappings,
				ExtraMounts:       extraMounts,
			},
		},
	}

	return &c, nil
}

func buildKubeadmConfigV1beta3(conf BuildKindConfig) (*kubeadmv1beta3.ClusterConfiguration, error) {
	c := kubeadmv1beta3.ClusterConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterConfiguration",
			APIVersion: "kubeadm.k8s.io/v1beta3",
		},
		Etcd: kubeadmv1beta3.Etcd{
			Local: &kubeadmv1beta3.LocalEtcd{
				DataDir: "/var/lib/etcd",
			},
		},
	}

	if len(conf.EtcdExtraArgs) > 0 {
		c.Etcd.Local.ExtraArgs = map[string]string{}
		for _, arg := range conf.EtcdExtraArgs {
			c.Etcd.Local.ExtraArgs[arg.Key] = arg.Value
		}
	}

	if len(conf.EtcdExtraVolumes) > 0 {
		return nil, fmt.Errorf("etcd extra volumes are not supported")
	}

	if len(conf.ApiserverExtraArgs) > 0 {
		c.APIServer.ExtraArgs = map[string]string{}
		for _, arg := range conf.ApiserverExtraArgs {
			c.APIServer.ExtraArgs[arg.Key] = arg.Value
		}
	}

	if len(conf.ApiserverExtraVolumes) > 0 {
		c.APIServer.ExtraVolumes = []kubeadmv1beta3.HostPathMount{}
		for _, vol := range conf.ApiserverExtraVolumes {
			c.APIServer.ExtraVolumes = append(c.APIServer.ExtraVolumes, kubeadmv1beta3.HostPathMount{
				Name:      vol.Name,
				HostPath:  fmt.Sprintf("/var/components/apiserver%s", vol.MountPath),
				MountPath: vol.MountPath,
				ReadOnly:  vol.ReadOnly,
				PathType:  corev1.HostPathType(vol.PathType),
			})
		}
	}

	if len(conf.ControllerManagerExtraArgs) > 0 {
		c.ControllerManager.ExtraArgs = map[string]string{}
		for _, arg := range conf.ControllerManagerExtraArgs {
			c.ControllerManager.ExtraArgs[arg.Key] = arg.Value
		}
	}

	if len(conf.ControllerManagerExtraVolumes) > 0 {
		c.ControllerManager.ExtraVolumes = []kubeadmv1beta3.HostPathMount{}
		for _, vol := range conf.ControllerManagerExtraVolumes {
			c.ControllerManager.ExtraVolumes = append(c.ControllerManager.ExtraVolumes, kubeadmv1beta3.HostPathMount{
				Name:      vol.Name,
				HostPath:  fmt.Sprintf("/var/components/controller-manager%s", vol.MountPath),
				MountPath: vol.MountPath,
				ReadOnly:  vol.ReadOnly,
				PathType:  corev1.HostPathType(vol.PathType),
			})
		}
	}

	if len(conf.SchedulerExtraArgs) > 0 {
		c.Scheduler.ExtraArgs = map[string]string{}
		for _, arg := range conf.SchedulerExtraArgs {
			c.Scheduler.ExtraArgs[arg.Key] = arg.Value
		}
	}

	if len(conf.SchedulerExtraVolumes) > 0 {
		c.Scheduler.ExtraVolumes = []kubeadmv1beta3.HostPathMount{}
		for _, vol := range conf.SchedulerExtraVolumes {
			c.Scheduler.ExtraVolumes = append(c.Scheduler.ExtraVolumes, kubeadmv1beta3.HostPathMount{
				Name:      vol.Name,
				HostPath:  fmt.Sprintf("/var/components/scheduler%s", vol.MountPath),
				MountPath: vol.MountPath,
				ReadOnly:  vol.ReadOnly,
				PathType:  corev1.HostPathType(vol.PathType),
			})
		}
	}

	return &c, nil
}
