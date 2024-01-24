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

package internalversion

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KwokctlConfiguration provides configuration for the Kwokctl.
type KwokctlConfiguration struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Options holds information about the default value.
	Options KwokctlConfigurationOptions
	// Components holds information about the components.
	Components []Component
	// ComponentsPatches holds information about the components patches.
	ComponentsPatches []ComponentPatches
	// Status holds information about the status.
	Status KwokctlConfigurationStatus
}

// KwokctlConfigurationStatus holds information about the status.
type KwokctlConfigurationStatus struct {
	// Version is the version of the kwokctl.
	Version string
}

// ExtraArgs holds information about the extra args.
type ExtraArgs struct {
	// Key is the key of the extra args.
	Key string
	// Value is the value of the extra args.
	Value string
}

// ComponentPatches holds information about the component patches.
type ComponentPatches struct {
	// Name is the name of the component.
	Name string
	// ExtraArgs is the extra args to be patched on the component.
	ExtraArgs []ExtraArgs
	// ExtraVolumes is the extra volumes to be patched on the component.
	ExtraVolumes []Volume
	// ExtraEnvs is the extra environment variables to be patched on the component.
	ExtraEnvs []Env
}

// KwokctlConfigurationOptions holds information about the options.
type KwokctlConfigurationOptions struct {
	// EnableCRDs is a list of CRDs to enable.
	EnableCRDs []string

	// KubeApiserverPort is the port to expose apiserver.
	KubeApiserverPort uint32

	// Runtime is the runtime to use.
	Runtime string

	// Runtimes is a list of alternate runtimes. When Runtime is empty,
	// the availability of the runtimes in the list is checked one by one
	// and set to Runtime
	Runtimes []string

	// PrometheusPort is the port to expose Prometheus metrics.
	PrometheusPort uint32

	// JaegerPort is the port to expose Jaeger UI.
	JaegerPort uint32

	// JaegerOtlpGrpcPort is the port to expose OTLP GRPC collector.
	JaegerOtlpGrpcPort uint32

	// KwokVersion is the version of Kwok to use.
	KwokVersion string

	// KubeVersion is the version of Kubernetes to use.
	KubeVersion string

	// EtcdVersion is the version of Etcd to use.
	EtcdVersion string

	// DashboardVersion is the version of Kubernetes dashboard to use.
	DashboardVersion string

	// PrometheusVersion is the version of Prometheus to use.
	PrometheusVersion string

	// JaegerVersion is the version of Jaeger to use.
	JaegerVersion string

	// MetricsServerVersion is the version of metrics-server to use.
	MetricsServerVersion string

	// DockerComposeVersion is the version of docker-compose to use.
	DockerComposeVersion string

	// KindVersion is the version of kind to use.
	KindVersion string

	// SecurePort is the apiserver port on which to serve HTTPS with authentication and authorization.
	// is not available before Kubernetes 1.13.0
	SecurePort bool

	// QuietPull is the flag to quiet the pull.
	QuietPull bool

	// KubeSchedulerConfig is the configuration path for kube-scheduler.
	KubeSchedulerConfig string

	// DisableKubeScheduler is the flag to disable kube-scheduler.
	DisableKubeScheduler bool

	// DisableKubeControllerManager is the flag to disable kube-controller-manager.
	DisableKubeControllerManager bool

	// EnableMetricsServer is the flag to enable metrics-server.
	EnableMetricsServer bool

	// EtcdImage is the image of etcd.
	EtcdImage string

	// KubeApiserverImage is the image of kube-apiserver.
	KubeApiserverImage string

	// KubeControllerManagerImage is the image of kube-controller-manager.
	KubeControllerManagerImage string

	// KubeSchedulerImage is the image of kube-scheduler.
	KubeSchedulerImage string

	// KwokControllerImage is the image of Kwok.
	KwokControllerImage string

	// DashboardImage is the image of dashboard.
	DashboardImage string

	// PrometheusImage is the image of Prometheus.
	PrometheusImage string

	// JaegerImage is the image of Jaeger
	JaegerImage string

	// MetricsServerImage is the image of metrics-server.
	MetricsServerImage string

	// KindNodeImage is the image of kind node.
	KindNodeImage string

	// BinSuffix is the suffix of the all binary.
	// On Windows is .exe
	BinSuffix string

	// KubeApiserverBinary is the binary of kube-apiserver.
	KubeApiserverBinary string

	// KubeControllerManagerBinary is the binary of kube-controller-manager.
	KubeControllerManagerBinary string

	// KubeSchedulerBinary is the binary of kube-scheduler.
	KubeSchedulerBinary string

	// KubectlBinary is the binary of kubectl.
	KubectlBinary string

	// EtcdctlBinary is the binary of etcdctl.
	EtcdctlBinary string

	// EtcdBinary is the binary of etcd.
	EtcdBinary string

	// EtcdBinaryTar is the tar of the binary of etcd.
	// Deprecated: Use EtcdBinary instead
	EtcdBinaryTar string

	// KwokControllerBinary is the binary of kwok.
	KwokControllerBinary string

	// TODO: Add dashboard binary
	// // DashboardBinary is the binary of dashboard.
	// DashboardBinary string

	// PrometheusBinary  is the binary of Prometheus.
	PrometheusBinary string

	// PrometheusBinaryTar is the tar of binary of Prometheus.
	// Deprecated: Use PrometheusBinary instead
	PrometheusBinaryTar string

	// JaegerBinary  is the binary of Jaeger.
	JaegerBinary string

	// JaegerBinaryTar is the tar of binary of Jaeger.
	// Deprecated: Use JaegerBinary instead
	JaegerBinaryTar string

	// MetricsServerBinary is the binary of metrics-server.
	MetricsServerBinary string

	// DockerComposeBinary is the binary of Docker compose.
	DockerComposeBinary string

	// KindBinary is the binary of kind.
	KindBinary string

	// KubeFeatureGates is a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes.
	KubeFeatureGates string

	// KubeRuntimeConfig is a set of key=value pairs that enable or disable built-in APIs.
	KubeRuntimeConfig string

	// KubeAuditPolicy is path to the file that defines the audit policy configuration
	KubeAuditPolicy string

	// KubeAuthorization is the flag to enable authorization on secure port.
	KubeAuthorization bool

	// KubeAdmission is the flag to enable admission for kube-apiserver.
	KubeAdmission bool

	// EtcdPeerPort is etcd peer port in the binary runtime
	EtcdPeerPort uint32

	// EtcdPort is etcd port in the binary runtime
	EtcdPort uint32

	// KubeControllerManagerPort is kube-controller-manager port in the binary runtime
	KubeControllerManagerPort uint32

	// KubeSchedulerPort is kube-scheduler port in the binary runtime
	KubeSchedulerPort uint32

	// DashboardPort is dashboard port that is exposed to the host.
	DashboardPort uint32

	// KwokControllerPort is kwok-controller port that is exposed to the host.
	KwokControllerPort uint32

	// MetricsServerPort is metrics-server port that is exposed to the host.
	MetricsServerPort uint32

	// CacheDir is the directory of the cache.
	CacheDir string

	// KubeControllerManagerNodeMonitorPeriodMilliseconds is --node-monitor-period for kube-controller-manager.
	KubeControllerManagerNodeMonitorPeriodMilliseconds int64

	// KubeControllerManagerNodeMonitorGracePeriodMilliseconds is --node-monitor-grace-period for kube-controller-manager.
	KubeControllerManagerNodeMonitorGracePeriodMilliseconds int64

	// NodeStatusUpdateFrequencyMilliseconds is --node-status-update-frequency for kwok like kubelet.
	NodeStatusUpdateFrequencyMilliseconds int64

	// NodeLeaseDurationSeconds is the duration the Kubelet will set on its corresponding Lease.
	NodeLeaseDurationSeconds uint

	// BindAddress is the address to bind to.
	BindAddress string

	// KubeApiserverCertSANs sets extra Subject Alternative Names for the API Server signing cert.
	KubeApiserverCertSANs []string

	// DisableQPSLimits specifies whether to disable QPS limits for components.
	DisableQPSLimits bool
}

// Component is a component of the cluster.
type Component struct {
	// Name of the component specified as a DNS_LABEL.
	// Each component must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string

	// Links is a set of links for the component.
	Links []string

	// Binary is the binary of the component.
	Binary string

	// Image is the image of the component.
	Image string

	// Command is Entrypoint array. Not executed within a shell. Only works with Image.
	Command []string

	// User is the user for the component.
	User string

	// Args is Arguments to the entrypoint.
	Args []string

	// WorkDir is component's working directory.
	WorkDir string

	// Ports is list of ports to expose from the component.
	Ports []Port

	// Envs is list of environment variables to set in the component.
	Envs []Env

	// Volumes is a list of named volumes that can be mounted by containers belonging to the component.
	Volumes []Volume

	// Metric is the metric of the component.
	Metric *ComponentMetric

	// MetricsDiscovery is the metrics discovery of the component.
	MetricsDiscovery *ComponentMetric

	// Version is the version of the component.
	Version string
}

// Env represents an environment variable present in a Container.
type Env struct {
	// Name of the environment variable.
	Name string

	// Value is using the previously defined environment variables in the component.
	Value string
}

// Port represents a network port in a single component.
type Port struct {
	// Name for the port that can be referred to by components.
	Name string
	// Port is number of port to expose on the component's IP address.
	// This must be a valid port number, 0 < x < 65536.
	Port uint32
	// HostPort is number of port to expose on the host.
	// If specified, this must be a valid port number, 0 < x < 65536.
	HostPort uint32
	// Protocol for port. Must be UDP, TCP, or SCTP.
	Protocol Protocol
}

// ComponentMetric represents a metric of a component.
type ComponentMetric struct {
	// Scheme is the scheme of the metric.
	Scheme string
	// Host is the host of the metric.
	Host string
	// Path is the path of the metric.
	Path string

	// CertPath is the cert path of the metric.
	CertPath string
	// KeyPath is the key path of the metric.
	KeyPath string
	// InsecureSkipVerify is the flag to skip verify the metric.
	InsecureSkipVerify bool
}

// Protocol defines network protocols supported for things like component ports.
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
	Name string
	// Mounted read-only if true, read-write otherwise (false or unspecified).
	ReadOnly bool
	// HostPath represents a pre-existing file or directory on the host machine that is directly exposed to the container.
	HostPath string
	// MountPath within the container at which the volume should be mounted.
	MountPath string
	// PathType is the type of the HostPath.
	PathType HostPathType
}

// HostPathType represents the type of storage used for HostPath volumes.
type HostPathType string

// Constants for HostPathType.
const (
	// For backwards compatible, leave it empty if unset
	HostPathUnset HostPathType = ""
	// If nothing exists at the given path, an empty directory will be created there
	// as needed with file mode 0755, having the same group and ownership with Kubelet.
	HostPathDirectoryOrCreate HostPathType = "DirectoryOrCreate"
	// A directory must exist at the given path
	HostPathDirectory HostPathType = "Directory"
	// If nothing exists at the given path, an empty file will be created there
	// as needed with file mode 0644, having the same group and ownership with Kubelet.
	HostPathFileOrCreate HostPathType = "FileOrCreate"
	// A file must exist at the given path
	HostPathFile HostPathType = "File"
	// A UNIX socket must exist at the given path
	HostPathSocket HostPathType = "Socket"
	// A character device must exist at the given path
	HostPathCharDev HostPathType = "CharDevice"
	// A block device must exist at the given path
	HostPathBlockDev HostPathType = "BlockDevice"
)
