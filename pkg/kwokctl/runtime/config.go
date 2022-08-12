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

package runtime

import (
	"context"
	"io"
	"time"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
)

type Config struct {
	Name           string `json:"name,omitempty"`
	ApiserverPort  uint32 `json:"apiserver_port,omitempty"`
	Workdir        string `json:"workdir,omitempty"`
	Runtime        string `json:"runtime,omitempty"`
	PrometheusPort uint32 `json:"prometheus_port,omitempty"`

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

	// Cache directory
	CacheDir string `json:"cache_dir,omitempty"`

	// For docker-compose and binary
	SecretPort bool `json:"secret_port,omitempty"`

	// Pull image
	QuietPull bool `json:"quiet_pull,omitempty"`

	// Feature gates of Kubernetes
	FeatureGates string `json:"kube_feature_gates,omitempty"`

	// Runtime config of Kubernetes
	RuntimeConfig string `json:"kube_runtime_config,omitempty"`
}

type Runtime interface {
	// Init the config of cluster
	Init(ctx context.Context, conf Config) error

	// Config return the config of cluster
	Config() (*Config, error)

	// Install the cluster
	Install(ctx context.Context) error

	// Uninstall the cluster
	Uninstall(ctx context.Context) error

	// Up start the cluster
	Up(ctx context.Context) error

	// Down stop the cluster
	Down(ctx context.Context) error

	// Start start a container
	Start(ctx context.Context, name string) error

	// Stop stop a container
	Stop(ctx context.Context, name string) error

	// Ready check the cluster is ready
	Ready(ctx context.Context) (bool, error)

	// WaitReady wait the cluster is ready
	WaitReady(ctx context.Context, timeout time.Duration) error

	// InHostKubeconfig return the kubeconfig in host
	InHostKubeconfig() (string, error)

	// Kubectl command
	Kubectl(ctx context.Context, stm utils.IOStreams, args ...string) error

	// KubectlInCluster command in cluster
	KubectlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error

	// Logs logs of a component
	Logs(ctx context.Context, name string, out io.Writer) error

	// LogsFollow follow logs of a component with follow
	LogsFollow(ctx context.Context, name string, out io.Writer) error

	// ListBinaries list binaries in the cluster
	ListBinaries(ctx context.Context, actual bool) ([]string, error)

	// ListImages list images in the cluster
	ListImages(ctx context.Context, actual bool) ([]string, error)
}
