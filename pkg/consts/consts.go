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

// Package consts defines the constants for building the project.
package consts

// The following constants are used to build the project.
var (
	ProjectName = "kwok"
	ConfigName  = "kwok.yaml"

	// Version is the version of the project.
	// It will be overwritten during the `make build` process.
	Version      = "0.4.0"
	BinaryPrefix = "https://github.com/kubernetes-sigs/kwok/releases/download"
	ImagePrefix  = "registry.k8s.io/kwok"

	// PreRelease is the pre-release version of the project.
	// It will be overwritten during the `make build` process.
	PreRelease = "alpha"

	// KubeVersion is the version of Kubernetes.
	// It will be overwritten during the `make build` process.
	KubeVersion      = "1.28.0"
	KubeBinaryPrefix = "https://dl.k8s.io/release"
	KubeImagePrefix  = "registry.k8s.io"

	EtcdBinaryPrefix = "https://github.com/etcd-io/etcd/releases/download"

	// DockerComposeVersion
	// Deprecated: will be removed in the future.
	DockerComposeVersion      = "2.17.2"
	DockerComposeBinaryPrefix = "https://github.com/docker/compose/releases/download"

	KindVersion         = "0.19.0"
	KindBinaryPrefix    = "https://github.com/kubernetes-sigs/kind/releases/download"
	KindNodeImagePrefix = "docker.io/kindest"

	DashboardVersion      = "2.7.0"
	DashboardBinaryPrefix = ""
	DashboardImagePrefix  = "docker.io/kubernetesui"

	PrometheusVersion      = "2.44.0"
	PrometheusBinaryPrefix = "https://github.com/prometheus/prometheus/releases/download"
	PrometheusImagePrefix  = "docker.io/prom"

	JaegerVersion      = "1.45.0"
	JaegerBinaryPrefix = "https://github.com/jaegertracing/jaeger/releases/download"
	JaegerImagePrefix  = "docker.io/jaegertracing"

	DefaultUnlimitedQPS   = 5000.0
	DefaultUnlimitedBurst = 10000
)

// The following runtime is provided.
const (
	RuntimeTypeKind       = "kind"
	RuntimeTypeKindPodman = RuntimeTypeKind + "-" + RuntimeTypePodman
	RuntimeTypeDocker     = "docker"
	RuntimeTypeNerdctl    = "nerdctl"
	RuntimeTypePodman     = "podman"
	RuntimeTypeBinary     = "binary"
)

// The following components is provided.
const (
	ComponentEtcd                  = "etcd"
	ComponentKubeApiserver         = "kube-apiserver"
	ComponentKubeControllerManager = "kube-controller-manager"
	ComponentKubeScheduler         = "kube-scheduler"
	ComponentKwokController        = "kwok-controller"
	ComponentDashboard             = "dashboard"
	ComponentPrometheus            = "prometheus"
	ComponentJaeger                = "jaeger"
)
