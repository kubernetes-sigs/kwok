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
	Version      = "0.7.0"
	BinaryPrefix = "https://github.com/kubernetes-sigs/kwok/releases/download"
	ImagePrefix  = "registry.k8s.io/kwok"

	// PreRelease is the pre-release version of the project.
	// It will be overwritten during the `make build` process.
	PreRelease = "alpha"

	// KubeVersion is the version of Kubernetes.
	// It will be overwritten during the `make build` process.
	KubeVersion                = "1.33.0"
	KubeBinaryPrefix           = "https://dl.k8s.io/release"
	KubeBinaryUnofficialPrefix = "https://github.com/kwok-ci/k8s/releases/download"
	KubeImagePrefix            = "registry.k8s.io"

	EtcdBinaryPrefix = "https://github.com/etcd-io/etcd/releases/download"

	KectlVersion      = "0.0.8"
	KectlBinaryPrefix = "https://github.com/kwok-ci/kectl/releases/download"

	KindVersion         = "0.23.0"
	KindBinaryPrefix    = "https://github.com/kubernetes-sigs/kind/releases/download"
	KindNodeImagePrefix = "docker.io/kindest"

	// DashboardVersion last available version is 2.7.0
	// Deprecated: Use DashboardWebVersion and DashboardApiVersion instead
	DashboardVersion               = "2.7.0"
	DashboardWebVersion            = "1.6.2"
	DashboardApiVersion            = "1.11.1"
	DashboardMetricsScraperVersion = "1.2.2"
	DashboardBinaryPrefix          = ""
	DashboardImagePrefix           = "docker.io/kubernetesui"

	PrometheusVersion      = "2.53.0"
	PrometheusBinaryPrefix = "https://github.com/prometheus/prometheus/releases/download"
	PrometheusImagePrefix  = "docker.io/prom"

	JaegerVersion      = "1.58.1"
	JaegerBinaryPrefix = "https://github.com/jaegertracing/jaeger/releases/download"
	JaegerImagePrefix  = "docker.io/jaegertracing"

	MetricsServerVersion      = "0.7.1"
	MetricsServerBinaryPrefix = "https://github.com/kubernetes-sigs/metrics-server/releases/download"
	MetricsServerImagePrefix  = "registry.k8s.io/metrics-server"

	DefaultUnlimitedQPS   = 5000.0
	DefaultUnlimitedBurst = 10000
)

// The following runtime is provided.
const (
	// RuntimeTypeBinary is the binary runtime.
	RuntimeTypeBinary = "binary"

	// Container runtime type, will create a container for each component.

	// RuntimeTypeDocker is the docker runtime.
	RuntimeTypeDocker = "docker"
	// RuntimeTypePodman is the podman runtime.
	RuntimeTypePodman = "podman"
	// RuntimeTypeNerdctl is the nerdctl runtime.
	RuntimeTypeNerdctl = "nerdctl"
	// RuntimeTypeLima is the lima runtime.
	RuntimeTypeLima = "lima"
	// RuntimeTypeFinch is the finch runtime.
	RuntimeTypeFinch = "finch"

	// Cluster runtime type, creates a cluster and deploys the components in the cluster.

	// RuntimeTypeKind is the kind runtime.
	RuntimeTypeKind = "kind"
	// RuntimeTypeKindPodman is the kind runtime with podman.
	RuntimeTypeKindPodman = RuntimeTypeKind + "-" + RuntimeTypePodman
	// RuntimeTypeKindNerdctl is the kind runtime with nerdctl.
	RuntimeTypeKindNerdctl = RuntimeTypeKind + "-" + RuntimeTypeNerdctl
	// RuntimeTypeKindLima is the kind runtime with lima.
	RuntimeTypeKindLima = RuntimeTypeKind + "-" + RuntimeTypeLima
	// RuntimeTypeKindFinch is the kind runtime with finch.
	RuntimeTypeKindFinch = RuntimeTypeKind + "-" + RuntimeTypeFinch
)

// The following components is provided.
const (
	ComponentEtcd                       = "etcd"
	ComponentKubeApiserver              = "kube-apiserver"
	ComponentKubeApiserverInsecureProxy = "kube-apiserver-insecure-proxy"
	ComponentKubeControllerManager      = "kube-controller-manager"
	ComponentKubeScheduler              = "kube-scheduler"
	ComponentKwokController             = "kwok-controller"
	ComponentDashboard                  = "dashboard"
	ComponentDashboardApi               = "dashboard-api"
	ComponentDashboardWeb               = "dashboard-web"
	ComponentDashboardMetricsScraper    = "dashboard-metrics-scraper"
	ComponentPrometheus                 = "prometheus"
	ComponentJaeger                     = "jaeger"
	ComponentMetricsServer              = "metrics-server"
)
