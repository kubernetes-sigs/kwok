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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KwokctlConfiguration provides configuration for the Kwokctl.
type KwokctlConfiguration struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Options holds information about the default value.
	Options KwokctlConfigurationOptions
}

type KwokctlConfigurationOptions struct {

	// KubeApiserverPort is the port to expose apiserver.
	KubeApiserverPort uint32

	// Runtime is the runtime to use.
	Runtime string

	// PrometheusPort is the port to expose Prometheus metrics.
	PrometheusPort uint32

	// KwokVersion is the version of Kwok to use.
	KwokVersion string

	// KubeVersion is the version of Kubernetes to use.
	KubeVersion string

	// EtcdVersion is the version of Etcd to use.
	EtcdVersion string

	// PrometheusVersion is the version of Prometheus to use.
	PrometheusVersion string

	// DockerComposeVersion is the version of docker-compose to use.
	DockerComposeVersion string

	// SecurePort is the apiserver port on which to serve HTTPS with authentication and authorization.
	SecurePort bool

	// QuietPull is the flag to quiet the pull.
	QuietPull bool

	// DisableKubeScheduler is the flag to disable kube-scheduler.
	DisableKubeScheduler bool

	// DisableKubeControllerManager is the flag to disable kube-controller-manager.
	DisableKubeControllerManager bool

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

	// PrometheusImage is the image of Prometheus.
	PrometheusImage string

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

	// EtcdBinary is the binary of etcd.
	EtcdBinary string

	// EtcdBinaryTar is the tar of the binary of etcd.
	EtcdBinaryTar string

	// KwokControllerBinary is the binary of kwok.
	KwokControllerBinary string

	// PrometheusBinary  is the binary of Prometheus.
	PrometheusBinary string

	// PrometheusBinaryTar is the tar of binary of Prometheus.
	PrometheusBinaryTar string

	// DockerComposeBinary is the binary of Docker compose.
	DockerComposeBinary string

	// Mode is several default parameter templates for clusters
	Mode string

	// KubeFeatureGates is a set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes.
	KubeFeatureGates string

	// KubeRuntimeConfig is a set of key=value pairs that enable or disable built-in APIs.
	KubeRuntimeConfig string

	// KubeAuditPolicy is path to the file that defines the audit policy configuration
	KubeAuditPolicy string

	// KubeAuthorization is the flag to enable authorization on secure port.
	KubeAuthorization bool

	// EtcdPeerPort is etcd peer port in the binary runtime
	EtcdPeerPort uint32

	// EtcdPort is etcd port in the binary runtime
	EtcdPort uint32

	// KubeControllerManagerPort is kube-controller-manager port in the binary runtime
	KubeControllerManagerPort uint32

	// KubeSchedulerPort is kube-scheduler port in the binary runtime
	KubeSchedulerPort uint32

	// KwokControllerPort is kube-controller port in the binary runtime
	KwokControllerPort uint32

	// CacheDir is the directory of the cache.
	CacheDir string
}
