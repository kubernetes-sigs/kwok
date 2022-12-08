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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
)

type Runtime interface {
	// SetConfig sets the config of cluster
	SetConfig(ctx context.Context, conf *internalversion.KwokctlConfigurationOptions) error

	// Save the config of cluster
	Save(ctx context.Context) error

	// Config return the config of cluster
	Config(ctx context.Context) (*internalversion.KwokctlConfigurationOptions, error)

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

	// EtcdctlInCluster command in cluster
	EtcdctlInCluster(ctx context.Context, stm utils.IOStreams, args ...string) error

	// Logs logs of a component
	Logs(ctx context.Context, name string, out io.Writer) error

	// LogsFollow follow logs of a component with follow
	LogsFollow(ctx context.Context, name string, out io.Writer) error

	// AuditLogs audit logs of apiserver
	AuditLogs(ctx context.Context, out io.Writer) error

	// AuditLogsFollow follow audit logs of apiserver
	AuditLogsFollow(ctx context.Context, out io.Writer) error

	// ListBinaries list binaries in the cluster
	ListBinaries(ctx context.Context) ([]string, error)

	// ListImages list images in the cluster
	ListImages(ctx context.Context) ([]string, error)

	// SnapshotSave save the snapshot of cluster
	SnapshotSave(ctx context.Context, path string) error

	// SnapshotRestore restore the snapshot of cluster
	SnapshotRestore(ctx context.Context, path string) error
}
