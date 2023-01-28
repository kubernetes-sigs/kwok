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

	Version      = "unknown"
	BinaryPrefix = "https://github.com/kubernetes-sigs/kwok/releases/download"
	ImagePrefix  = "registry.k8s.io/kwok"

	KubeVersion      = "unknown"
	KubeBinaryPrefix = "https://dl.k8s.io/release"
	KubeImagePrefix  = "registry.k8s.io"

	EtcdBinaryPrefix = "https://github.com/etcd-io/etcd/releases/download"

	DockerComposeVersion      = "2.13.0"
	DockerComposeBinaryPrefix = "https://github.com/docker/compose/releases/download"

	KindVersion      = "0.17.0"
	KindBinaryPrefix = "https://github.com/kubernetes-sigs/kind/releases/download"

	PrometheusVersion      = "2.41.0"
	PrometheusBinaryPrefix = "https://github.com/prometheus/prometheus/releases/download"
	PrometheusImagePrefix  = "docker.io/prom"

	KindNodeImagePrefix = "docker.io/kindest"
)

// The following runtime is provided.
const (
	RuntimeTypeKind    = "kind"
	RuntimeTypeDocker  = "docker"
	RuntimeTypeNerdctl = "nerdctl"
	RuntimeTypeBinary  = "binary"
)
