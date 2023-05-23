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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KwokctlConfigurationKind is the kind of the kwokctl configuration.
	KwokctlConfigurationKind = "KwokctlConfiguration"

	// ModeStableFeatureGateAndAPI is intended to reduce cluster configuration requirements
	// Disables all Alpha feature by default, as well as Beta feature that are not eventually GA
	ModeStableFeatureGateAndAPI = "StableFeatureGateAndAPI"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KwokctlConfiguration provides configuration for the Kwokctl.
type KwokctlConfiguration struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Options holds information about the default value.
	Options KwokctlConfigurationOptions `json:"options,omitempty"`
	// Components holds information about the components.
	Components []Component `json:"components,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// ComponentsPatches holds information about the components patches.
	ComponentsPatches []ComponentPatches `json:"componentsPatches,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

// ExtraArgs holds information about the extra args.
type ExtraArgs struct {
	// Key is the key of the extra args.
	Key string `json:"key"`
	// Value is the value of the extra args.
	Value string `json:"value"`
}

// ComponentPatches holds information about the component patches.
type ComponentPatches struct {
	// Name is the name of the component.
	Name string `json:"name"`
	// ExtraArgs is the extra args to be patched on the component.
	ExtraArgs []ExtraArgs `json:"extraArgs,omitempty"`
	// ExtraVolumes is the extra volumes to be patched on the component.
	ExtraVolumes []Volume `json:"extraVolumes,omitempty"`
}

// KwokctlConfigurationOptions holds information about the options.
type KwokctlConfigurationOptions struct {
	// KubeApiserverPort is the port to expose apiserver.
	// is the default value for flag --kube-apiserver-port and env KWOK_KUBE_APISERVER_PORT
	KubeApiserverPort uint32 `json:"kubeApiserverPort,omitempty"`

	// Runtime is the runtime to use.
	// is the default value for flag --runtime and env KWOK_RUNTIME
	Runtime string `json:"runtime,omitempty"`

	// Runtimes is a list of alternate runtimes. When Runtime is empty,
	// the availability of the runtimes in the list is checked one by one
	// and set to Runtime
	Runtimes []string `json:"runtimes,omitempty"`

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
	SecurePort *bool `json:"securePort,omitempty"`

	// QuietPull is the flag to quiet the pull.
	// is the default value for flag --quiet-pull and env KWOK_QUIET_PULL
	// +default=false
	QuietPull *bool `json:"quietPull,omitempty"`

	// KubeSchedulerConfig is the configuration path for kube-scheduler.
	// is the default value for flag --kube-scheduler-config and env KWOK_KUBE_SCHEDULER_CONFIG
	KubeSchedulerConfig string `json:"kubeSchedulerConfig,omitempty"`

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
	//+k8s:conversion-gen=false
	KubeImagePrefix string `json:"kubeImagePrefix,omitempty"`

	// EtcdImagePrefix is the prefix of the etcd image.
	// is the default value for env KWOK_ETCD_IMAGE_PREFIX
	//+k8s:conversion-gen=false
	EtcdImagePrefix string `json:"etcdImagePrefix,omitempty"`

	// KwokImagePrefix is the prefix of the kwok image.
	// is the default value for env KWOK_IMAGE_PREFIX
	//+k8s:conversion-gen=false
	KwokImagePrefix string `json:"kwokImagePrefix,omitempty"`

	// PrometheusImagePrefix is the prefix of the Prometheus image.
	// is the default value for env KWOK_PROMETHEUS_IMAGE_PREFIX
	//+k8s:conversion-gen=false
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
	// is the default value for flag --controller-image and env KWOK_CONTROLLER_IMAGE
	KwokControllerImage string `json:"kwokControllerImage,omitempty"`

	// PrometheusImage is the image of Prometheus.
	// is the default value for flag --prometheus-image and env KWOK_PROMETHEUS_IMAGE
	PrometheusImage string `json:"prometheusImage,omitempty"`

	// KindNodeImagePrefix is the prefix of the kind node image.
	// is the default value for env KWOK_KIND_NODE_IMAGE_PREFIX
	//+k8s:conversion-gen=false
	KindNodeImagePrefix string `json:"kindNodeImagePrefix,omitempty"`

	// KindNodeImage is the image of kind node.
	// is the default value for flag --kind-node-image and env KWOK_KIND_NODE_IMAGE
	KindNodeImage string `json:"kindNodeImage,omitempty"`

	// BinSuffix is the suffix of the all binary.
	// On Windows is .exe
	BinSuffix string `json:"binSuffix,omitempty"`

	// KubeBinaryPrefix is the prefix of the kubernetes binary.
	// is the default value for env KWOK_KUBE_BINARY_PREFIX
	//+k8s:conversion-gen=false
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
	//+k8s:conversion-gen=false
	EtcdBinaryPrefix string `json:"etcdBinaryPrefix,omitempty"`

	// EtcdBinary is the binary of etcd.
	// is the default value for flag --etcd-binary and env KWOK_ETCD_BINARY
	EtcdBinary string `json:"etcdBinary,omitempty"`

	// EtcdBinaryTar is the tar of the binary of etcd.
	// is the default value for env KWOK_ETCD_BINARY_TAR
	EtcdBinaryTar string `json:"etcdBinaryTar,omitempty"`

	// KwokBinaryPrefix is the prefix of the kwok binary.
	// is the default value for env KWOK_BINARY_PREFIX
	//+k8s:conversion-gen=false
	KwokBinaryPrefix string `json:"kwokBinaryPrefix,omitempty"`

	// KwokControllerBinary is the binary of kwok.
	// is the default value for flag --controller-binary and env KWOK_CONTROLLER_BINARY
	KwokControllerBinary string `json:"kwokControllerBinary,omitempty"`

	// PrometheusBinaryPrefix is the prefix of the Prometheus binary.
	// is the default value for env KWOK_PROMETHEUS_PREFIX
	//+k8s:conversion-gen=false
	PrometheusBinaryPrefix string `json:"prometheusBinaryPrefix,omitempty"`

	// PrometheusBinary  is the binary of Prometheus.
	// is the default value for flag --prometheus-binary and env KWOK_PROMETHEUS_BINARY
	PrometheusBinary string `json:"prometheusBinary,omitempty"`

	// PrometheusBinaryTar is the tar of binary of Prometheus.
	// is the default value for env KWOK_PROMETHEUS_BINARY_TAR
	PrometheusBinaryTar string `json:"prometheusBinaryTar,omitempty"`

	// DockerComposeBinaryPrefix is the binary of docker-compose.
	// is the default value for env KWOK_DOCKER_COMPOSE_BINARY_PREFIX
	//+k8s:conversion-gen=false
	DockerComposeBinaryPrefix string `json:"dockerComposeBinaryPrefix,omitempty"`

	// DockerComposeBinary is the binary of Docker compose.
	// is the default value for flag --docker-compose-binary and env KWOK_DOCKER_COMPOSE_BINARY
	DockerComposeBinary string `json:"dockerComposeBinary,omitempty"`

	// KindBinaryPrefix is the binary prefix of kind.
	// is the default value for env KWOK_KIND_BINARY_PREFIX
	//+k8s:conversion-gen=false
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

	// KubeAdmission is the flag to enable admission for kube-apiserver.
	// is the default value for flag --kube-admission and env KWOK_KUBE_ADMISSION
	// +default=false
	KubeAdmission *bool `json:"kubeAdmission,omitempty"`

	// EtcdPeerPort is etcd peer port in the binary runtime
	EtcdPeerPort uint32 `json:"etcdPeerPort,omitempty"`

	// EtcdPort is etcd port in the binary runtime
	EtcdPort uint32 `json:"etcdPort,omitempty"`

	// KubeControllerManagerPort is kube-controller-manager port in the binary runtime
	KubeControllerManagerPort uint32 `json:"kubeControllerManagerPort,omitempty"`

	// KubeSchedulerPort is kube-scheduler port in the binary runtime
	KubeSchedulerPort uint32 `json:"kubeSchedulerPort,omitempty"`

	// KwokControllerPort is kwok-controller port that is exposed to the host.
	// is the default value for flag --controller-port and env KWOK_CONTROLLER_PORT
	KwokControllerPort uint32 `json:"kwokControllerPort,omitempty"`

	// CacheDir is the directory of the cache.
	CacheDir string `json:"cacheDir,omitempty"`

	// KubeControllerManagerNodeMonitorPeriodMilliseconds is --node-monitor-period for kube-controller-manager.
	// +default=600000
	KubeControllerManagerNodeMonitorPeriodMilliseconds int64 `json:"kubeControllerManagerNodeMonitorPeriodMilliseconds,omitempty"`

	// KubeControllerManagerNodeMonitorGracePeriodMilliseconds is --node-monitor-grace-period for kube-controller-manager.
	// +default=3600000
	KubeControllerManagerNodeMonitorGracePeriodMilliseconds int64 `json:"kubeControllerManagerNodeMonitorGracePeriodMilliseconds,omitempty"`

	// NodeStatusUpdateFrequencyMilliseconds is --node-status-update-frequency for kwok like kubelet.
	// +default=1200000
	NodeStatusUpdateFrequencyMilliseconds int64 `json:"nodeStatusUpdateFrequencyMilliseconds,omitempty"`

	// NodeLeaseDurationSeconds is the duration the Kubelet will set on its corresponding Lease.
	// +default=1200
	NodeLeaseDurationSeconds uint `json:"nodeLeaseDurationSeconds,omitempty"`

	// BindAddress is the address to bind to.
	// +default="0.0.0.0"
	BindAddress string `json:"bindAddress,omitempty"`
}

// Component is a component of the cluster.
type Component struct {
	// Name of the component specified as a DNS_LABEL.
	// Each component must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name"`

	// Links is a set of links for the component.
	// +optional
	Links []string `json:"links,omitempty"`

	// Binary is the binary of the component.
	// +optional
	Binary string `json:"binary,omitempty"`

	// Image is the image of the component.
	// +optional
	Image string `json:"image,omitempty"`

	// Command is Entrypoint array. Not executed within a shell. Only works with Image.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args is Arguments to the entrypoint.
	// +optional
	Args []string `json:"args,omitempty"`

	// WorkDir is component's working directory.
	// +optional
	WorkDir string `json:"workDir,omitempty"`

	// Ports is list of ports to expose from the component.
	// +optional
	Ports []Port `json:"ports,omitempty"`

	// Envs is list of environment variables to set in the component.
	// +optional
	Envs []Env `json:"envs,omitempty"`

	// Volumes is a list of named volumes that can be mounted by containers belonging to the component.
	// +optional
	Volumes []Volume `json:"volumes,omitempty"`

	// Version is the version of the component.
	// +optional
	Version string `json:"version,omitempty"`
}

// Env represents an environment variable present in a Container.
type Env struct {
	// Name of the environment variable.
	Name string `json:"name"`

	// Value is using the previously defined environment variables in the component.
	// +optional
	// +default=""
	Value string `json:"value,omitempty"`
}

// Port represents a network port in a single component.
type Port struct {
	// Name for the port that can be referred to by components.
	// +optional
	Name string `json:"name,omitempty"`
	// Port is number of port to expose on the component's IP address.
	// This must be a valid port number, 0 < x < 65536.
	Port uint32 `json:"port"`
	// HostPort is number of port to expose on the host.
	// If specified, this must be a valid port number, 0 < x < 65536.
	// +optional
	HostPort uint32 `json:"hostPort,omitempty"`
	// Protocol for port. Must be UDP, TCP, or SCTP.
	// +optional
	// +default="TCP"
	Protocol Protocol `json:"protocol,omitempty"`
}

// Protocol defines network protocols supported for things like component ports.
// +enum
type Protocol string

const (
	// ProtocolTCP is the TCP protocol.
	ProtocolTCP Protocol = "TCP"
	// ProtocolUDP is the UDP protocol.
	ProtocolUDP Protocol = "UDP"
	// ProtocolSCTP is the SCTP protocol.
	ProtocolSCTP Protocol = "SCTP"
)

// Volume represents a volume that is accessible to the containers running in a component.
type Volume struct {
	// Name of the volume specified.
	// +optional
	Name string `json:"name,omitempty"`
	// Mounted read-only if true, read-write otherwise.
	// +optional
	ReadOnly *bool `json:"readOnly,omitempty"`
	// HostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.
	HostPath string `json:"hostPath,omitempty"`
	// MountPath within the container at which the volume should be mounted.
	MountPath string `json:"mountPath,omitempty"`
	// PathType is the type of the HostPath.
	PathType HostPathType `json:"pathType,omitempty"`
}

// HostPathType represents the type of storage used for HostPath volumes.
type HostPathType = corev1.HostPathType

// Constants for HostPathType.
const (
	HostPathUnset             HostPathType = corev1.HostPathUnset
	HostPathDirectoryOrCreate HostPathType = corev1.HostPathDirectoryOrCreate
	HostPathDirectory         HostPathType = corev1.HostPathDirectory
	HostPathFileOrCreate      HostPathType = corev1.HostPathFileOrCreate
	HostPathFile              HostPathType = corev1.HostPathFile
	HostPathSocket            HostPathType = corev1.HostPathSocket
	HostPathCharDev           HostPathType = corev1.HostPathCharDev
	HostPathBlockDev          HostPathType = corev1.HostPathBlockDev
)
