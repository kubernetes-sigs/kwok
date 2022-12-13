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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KwokctlConfigurationKind = "KwokctlConfiguration"

	// ModeStableFeatureGateAndAPI is intended to reduce cluster configuration requirements
	// Disables all Alpha feature by default, as well as Beta feature that are not eventually GA
	ModeStableFeatureGateAndAPI = "StableFeatureGateAndAPI"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KwokctlConfiguration provides configuration for the Kwokctl.
type KwokctlConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Options holds information about the default value.
	Options KwokctlConfigurationOptions `json:"options,omitempty"`
}

type KwokctlConfigurationOptions struct {

	// KubeApiserverPort is the port to expose apiserver.
	// is the default value for flag --kube-apiserver-port and env KWOK_KUBE_APISERVER_PORT
	KubeApiserverPort uint32 `json:"kubeApiserverPort,omitempty"`

	// Runtime is the runtime to use.
	// is the default value for flag --runtime and env KWOK_RUNTIME
	// +default="docker"
	Runtime string `json:"runtime,omitempty"`

	// PrometheusPort is the port to expose Prometheus metrics.
	// is the default value for flag --prometheus-port and env KWOK_PROMETHEUS_PORT
	PrometheusPort uint32 `json:"prometheusPort,omitempty"`

	// KwokVersion is the version of Kwok to use.
	// is the default value for env KWOK_VERSION
	KwokVersion string `json:"kwokVersion,omitempty"`

	// KubeVersion is the version of Kubernetes to use.
	// is the default value for env KWOK_KUBE_VERSION
	KubeVersion string `json:"kubeVersion,omitempty"`

	// EtcdVersion is the version of Etcd to use.
	// is the default value for env KWOK_ETCD_VERSION
	EtcdVersion string `json:"etcdVersion,omitempty"`

	// PrometheusVersion is the version of Prometheus to use.
	// is the default value for env KWOK_PROMETHEUS_VERSION
	PrometheusVersion string `json:"prometheusVersion,omitempty"`

	// DockerComposeVersion is the version of docker-compose to use.
	// is the default value for env KWOK_DOCKER_COMPOSE_VERSION
	DockerComposeVersion string `json:"dockerComposeVersion,omitempty"`

	// KindVersion is the version of kind to use.
	// is the default value for env KWOK_KIND_VERSION
	KindVersion string `json:"kindVersion,omitempty"`

	// SecurePort is the apiserver port on which to serve HTTPS with authentication and authorization.
	// is the default value for flag --secure-port and env KWOK_SECURE_PORT
	// +default=false
	SecurePort *bool `json:"securePort,omitempty"`

	// QuietPull is the flag to quiet the pull.
	// is the default value for flag --quiet-pull and env KWOK_QUIET_PULL
	// +default=false
	QuietPull *bool `json:"quietPull,omitempty"`

	// DisableKubeScheduler is the flag to disable kube-scheduler.
	// is the default value for flag --disable-kube-scheduler and env KWOK_DISABLE_KUBE_SCHEDULER
	// +default=false
	DisableKubeScheduler *bool `json:"disableKubeScheduler,omitempty"`

	// DisableKubeControllerManager is the flag to disable kube-controller-manager.
	// is the default value for flag --disable-kube-controller-manager and env KWOK_DISABLE_KUBE_CONTROLLER_MANAGER
	// +default=false
	DisableKubeControllerManager *bool `json:"disableKubeControllerManager,omitempty"`

	// KubeImagePrefix is the prefix of the kubernetes image.
	// is the default value for env KWOK_KUBE_IMAGE_PREFIX
	KubeImagePrefix string `json:"kubeImagePrefix,omitempty"`

	// EtcdImagePrefix is the prefix of the etcd image.
	// is the default value for env KWOK_ETCD_IMAGE_PREFIX
	EtcdImagePrefix string `json:"etcdImagePrefix,omitempty"`

	// KwokImagePrefix is the prefix of the kwok image.
	// is the default value for env KWOK_IMAGE_PREFIX
	KwokImagePrefix string `json:"kwokImagePrefix,omitempty"`

	// PrometheusImagePrefix is the prefix of the Prometheus image.
	// is the default value for env KWOK_PROMETHEUS_IMAGE_PREFIX
	PrometheusImagePrefix string `json:"prometheusImagePrefix,omitempty"`

	// EtcdImage is the image of etcd.
	// is the default value for flag --etcd-image and env KWOK_ETCD_IMAGE
	EtcdImage string `json:"etcdImage,omitempty"`

	// KubeApiserverImage is the image of kube-apiserver.
	// is the default value for flag --kube-apiserver-image and env KWOK_KUBE_APISERVER_IMAGE
	KubeApiserverImage string `json:"kubeApiserverImage,omitempty"`

	// KubeControllerManagerImage is the image of kube-controller-manager.
	// is the default value for flag --kube-controller-manager-image and env KWOK_KUBE_CONTROLLER_MANAGER_IMAGE
	KubeControllerManagerImage string `json:"kubeControllerManagerImage,omitempty"`

	// KubeSchedulerImage is the image of kube-scheduler.
	// is the default value for flag --kube-scheduler-image and env KWOK_KUBE_SCHEDULER_IMAGE
	KubeSchedulerImage string `json:"kubeSchedulerImage,omitempty"`

	// KwokControllerImage is the image of Kwok.
	// is the default value for flag --contoller-image and env KWOK_CONTROLLER_IMAGE
	KwokControllerImage string `json:"kwokControllerImage,omitempty"`

	// PrometheusImage is the image of Prometheus.
	// is the default value for flag --prometheus-image and env KWOK_PROMETHEUS_IMAGE
	PrometheusImage string `json:"prometheusImage,omitempty"`

	// KindNodeImagePrefix is the prefix of the kind node image.
	// is the default value for env KWOK_KIND_NODE_IMAGE_PREFIX
	KindNodeImagePrefix string `json:"kindNodeImagePrefix,omitempty"`

	// KindNodeImage is the image of kind node.
	// is the default value for flag --kind-node-image and env KWOK_KIND_NODE_IMAGE
	KindNodeImage string `json:"kindNodeImage,omitempty"`

	// BinSuffix is the suffix of the all binary.
	// On Windows is .exe
	BinSuffix string `json:"binSuffix,omitempty"`

	// KubeBinaryPrefix is the prefix of the kubernetes binary.
	// is the default value for env KWOK_KUBE_BINARY_PREFIX
	KubeBinaryPrefix string `json:"kubeBinaryPrefix,omitempty"`

	// KubeApiserverBinary is the binary of kube-apiserver.
	// is the default value for flag --apiserver-binary and env KWOK_KUBE_APISERVER_BINARY
	KubeApiserverBinary string `json:"kubeApiserverBinary,omitempty"`

	// KubeControllerManagerBinary is the binary of kube-controller-manager.
	// is the default value for flag --controller-manager-binary and env KWOK_KUBE_CONTROLLER_MANAGER_BINARY
	KubeControllerManagerBinary string `json:"kubeControllerManagerBinary,omitempty"`

	// KubeSchedulerBinary is the binary of kube-scheduler.
	// is the default value for flag --scheduler-binary and env KWOK_KUBE_SCHEDULER_BINARY
	KubeSchedulerBinary string `json:"kubeSchedulerBinary,omitempty"`

	// KubectlBinary is the binary of kubectl.
	// is the default value for env KWOK_KUBECTL_BINARY
	KubectlBinary string `json:"kubectlBinary,omitempty"`

	// EtcdBinaryPrefix is the prefix of the etcd binary.
	// is the default value for env KWOK_ETCD_BINARY_PREFIX
	EtcdBinaryPrefix string `json:"etcdBinaryPrefix,omitempty"`

	// EtcdBinary is the binary of etcd.
	// is the default value for flag --etcd-binary and env KWOK_ETCD_BINARY
	EtcdBinary string `json:"etcdBinary,omitempty"`

	// EtcdBinaryTar is the tar of the binary of etcd.
	// is the default value for env KWOK_ETCD_BINARY_TAR
	EtcdBinaryTar string `json:"etcdBinaryTar,omitempty"`

	// KwokBinaryPrefix is the prefix of the kwok binary.
	// is the default value for env KWOK_BINARY_PREFIX
	KwokBinaryPrefix string `json:"kwokBinaryPrefix,omitempty"`

	// KwokControllerBinary is the binary of kwok.
	// is the default value for flag --controller-binary and env KWOK_CONTROLLER_BINARY
	KwokControllerBinary string `json:"kwokControllerBinary,omitempty"`

	// PrometheusBinaryPrefix is the prefix of the Prometheus binary.
	// is the default value for env KWOK_PROMETHEUS_PREFIX
	PrometheusBinaryPrefix string `json:"prometheusBinaryPrefix,omitempty"`

	// PrometheusBinary  is the binary of Prometheus.
	// is the default value for flag --prometheus-binary and env KWOK_PROMETHEUS_BINARY
	PrometheusBinary string `json:"prometheusBinary,omitempty"`

	// PrometheusBinaryTar is the tar of binary of Prometheus.
	// is the default value for env KWOK_PROMETHEUS_BINARY_TAR
	PrometheusBinaryTar string `json:"prometheusBinaryTar,omitempty"`

	// DockerComposeBinaryPrefix is the binary of docker-compose.
	// is the default value for env KWOK_DOCKER_COMPOSE_BINARY_PREFIX
	DockerComposeBinaryPrefix string `json:"dockerComposeBinaryPrefix,omitempty"`

	// DockerComposeBinary is the binary of Docker compose.
	// is the default value for flag --docker-compose-binary and env KWOK_DOCKER_COMPOSE_BINARY
	DockerComposeBinary string `json:"dockerComposeBinary,omitempty"`

	// KindBinaryPrefix is the binary prefix of kind.
	// is the default value for env KWOK_KIND_BINARY_PREFIX
	KindBinaryPrefix string `json:"kindBinaryPrefix,omitempty"`

	// KindBinary is the binary of kind.
	// is the default value for flag --kind-binary and env KWOK_KIND_BINARY
	KindBinary string `json:"kindBinary,omitempty"`

	// Mode is several default parameter templates for clusters
	// is the default value for env KWOK_MODE
	Mode string `json:"mode,omitempty"`

	// KubeFeatureGates is a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes.
	// is the default value for flag --kube-feature-gates and env KWOK_KUBE_FEATURE_DATES
	KubeFeatureGates string `json:"kubeFeatureGates,omitempty"`

	// KubeRuntimeConfig is a set of key=value pairs that enable or disable built-in APIs.
	// is the default value for flag --kube-runtime-config and env KWOK_KUBE_RUNTIME_CONFIG
	KubeRuntimeConfig string `json:"kubeRuntimeConfig,omitempty"`

	// KubeAuditPolicy is path to the file that defines the audit policy configuration
	// is the default value for flag --kube-audit-policy and env KWOK_KUBE_AUDIT_POLICY
	KubeAuditPolicy string `json:"kubeAuditPolicy,omitempty"`

	// KubeAuthorization is the flag to enable authorization on secure port.
	// is the default value for flag --kube-authorization and env KWOK_KUBE_AUTHORIZATION
	// +default=false
	KubeAuthorization *bool `json:"kubeAuthorization,omitempty"`

	// EtcdPeerPort is etcd peer port in the binary runtime
	EtcdPeerPort uint32 `json:"etcdPeerPort,omitempty"`

	// EtcdPort is etcd port in the binary runtime
	EtcdPort uint32 `json:"etcdPort,omitempty"`

	// KubeControllerManagerPort is kube-controller-manager port in the binary runtime
	KubeControllerManagerPort uint32 `json:"kubeControllerManagerPort,omitempty"`

	// KubeSchedulerPort is kube-scheduler port in the binary runtime
	KubeSchedulerPort uint32 `json:"kubeSchedulerPort,omitempty"`

	// KwokControllerPort is kube-controller port in the binary runtime
	KwokControllerPort uint32 `json:"kwokControllerPort,omitempty"`

	// CacheDir is the directory of the cache.
	CacheDir string `json:"cacheDir,omitempty"`
}
