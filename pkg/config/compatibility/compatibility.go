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

// Package compatibility provides compatible for old version of kwokctl.
package compatibility

import (
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

// Config is the old configuration for kwokctl.
type Config struct {
	Name                      string `json:"name,omitempty"`
	Workdir                   string `json:"workdir,omitempty"`
	Runtime                   string `json:"runtime,omitempty"`
	EtcdPort                  uint32 `json:"etcd_port,omitempty"`
	EtcdPeerPort              uint32 `json:"etcd_peer_port,omitempty"`
	KubeApiserverPort         uint32 `json:"kube_apiserver_port,omitempty"`
	KubeControllerManagerPort uint32 `json:"kube_controller_manager_port,omitempty"`
	KubeSchedulerPort         uint32 `json:"kube_scheduler_port,omitempty"`
	KwokControllerPort        uint32 `json:"kwok_controller_port,omitempty"`
	PrometheusPort            uint32 `json:"prometheus_port,omitempty"`

	// For docker-compose
	EtcdImage                  string `json:"etcd_image,omitempty"`
	KubeApiserverImage         string `json:"kube_apiserver_image,omitempty"`
	KubeControllerManagerImage string `json:"kube_controller_manager_image,omitempty"`
	KubeSchedulerImage         string `json:"kube_scheduler_image,omitempty"`
	KwokControllerImage        string `json:"kwok_controller_image,omitempty"`
	PrometheusImage            string `json:"prometheus_image,omitempty"`

	// For kind
	KindNodeImage string `json:"kind_node_image,omitempty"`

	// For binary
	KubeApiserverBinary         string `json:"kube_apiserver_binary,omitempty"`
	KubeControllerManagerBinary string `json:"kube_controller_manager_binary,omitempty"`
	KubeSchedulerBinary         string `json:"kube_scheduler_binary,omitempty"`
	KwokControllerBinary        string `json:"kwok_controller_binary,omitempty"`
	EtcdBinary                  string `json:"etcd_binary,omitempty"`
	EtcdBinaryTar               string `json:"etcd_binary_tar,omitempty"`
	PrometheusBinary            string `json:"prometheus_binary,omitempty"`
	PrometheusBinaryTar         string `json:"prometheus_binary_tar,omitempty"`
	DockerComposeBinary         string `json:"docker_compose_binary,omitempty"`
	KubectlBinary               string `json:"kubectl_binary,omitempty"`

	// Cache directory
	CacheDir string `json:"cache_dir,omitempty"`

	// For docker-compose and binary
	SecretPort bool `json:"secret_port,omitempty"`

	// Pull image
	QuietPull bool `json:"quiet_pull,omitempty"`

	// Disable kube components
	DisableKubeScheduler         bool `json:"disable_kube_scheduler,omitempty"`
	DisableKubeControllerManager bool `json:"disable_kube_controller_manager,omitempty"`

	// Feature gates of Kubernetes
	FeatureGates string `json:"kube_feature_gates,omitempty"`

	// For audit log
	AuditPolicy string `json:"audit_policy,omitempty"`

	// Enable authorization on secure port
	Authorization bool `json:"authorization,omitempty"`

	// Runtime config of Kubernetes
	RuntimeConfig string `json:"kube_runtime_config,omitempty"`

	// DisableContextAutoSwitch is the flag to disable context auto switch.
	DisableContextAutoSwitch bool
}

// Convert_Config_To_internalversion_KwokctlConfiguration converts a Config to an internalversion.KwokctlConfiguration.
//
//nolint:revive
func Convert_Config_To_internalversion_KwokctlConfiguration(in *Config) (*internalversion.KwokctlConfiguration, bool) {
	if in.Name == "" || in.Workdir == "" || in.Runtime == "" {
		return nil, false
	}

	out := internalversion.KwokctlConfiguration{}
	out.Options.Runtime = in.Runtime
	out.Options.EtcdPort = in.EtcdPort
	out.Options.EtcdPeerPort = in.EtcdPeerPort
	out.Options.KubeApiserverPort = in.KubeApiserverPort
	out.Options.KubeControllerManagerPort = in.KubeControllerManagerPort
	out.Options.KubeSchedulerPort = in.KubeSchedulerPort
	out.Options.KwokControllerPort = in.KwokControllerPort
	out.Options.PrometheusPort = in.PrometheusPort
	out.Options.EtcdImage = in.EtcdImage
	out.Options.KubeApiserverImage = in.KubeApiserverImage
	out.Options.KubeControllerManagerImage = in.KubeControllerManagerImage
	out.Options.KubeSchedulerImage = in.KubeSchedulerImage
	out.Options.KwokControllerImage = in.KwokControllerImage
	out.Options.PrometheusImage = in.PrometheusImage
	out.Options.KindNodeImage = in.KindNodeImage
	out.Options.KubeApiserverBinary = in.KubeApiserverBinary
	out.Options.KubeControllerManagerBinary = in.KubeControllerManagerBinary
	out.Options.KubeSchedulerBinary = in.KubeSchedulerBinary
	out.Options.KwokControllerBinary = in.KwokControllerBinary
	out.Options.EtcdBinary = in.EtcdBinary
	out.Options.EtcdBinaryTar = in.EtcdBinaryTar
	out.Options.PrometheusBinary = in.PrometheusBinary
	out.Options.PrometheusBinaryTar = in.PrometheusBinaryTar
	out.Options.DockerComposeBinary = in.DockerComposeBinary
	out.Options.KubectlBinary = in.KubectlBinary
	out.Options.CacheDir = in.CacheDir
	out.Options.SecurePort = in.SecretPort
	out.Options.QuietPull = in.QuietPull
	out.Options.DisableKubeScheduler = in.DisableKubeScheduler
	out.Options.DisableKubeControllerManager = in.DisableKubeControllerManager
	out.Options.KubeFeatureGates = in.FeatureGates
	out.Options.KubeAuditPolicy = in.AuditPolicy
	out.Options.KubeAuthorization = in.Authorization
	out.Options.KubeRuntimeConfig = in.RuntimeConfig
	out.Options.DisableContextAutoSwitch = in.DisableContextAutoSwitch

	return &out, true
}
