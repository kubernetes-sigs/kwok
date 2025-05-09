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
	"sigs.k8s.io/kwok/pkg/utils/client"
)

// Runtime is the interface for a runtime.
type Runtime interface {
	// Available checks whether the runtime is available.
	Available(ctx context.Context) error

	// SetConfig sets the config of cluster
	SetConfig(ctx context.Context, conf *internalversion.KwokctlConfiguration) error

	// Save the config of cluster
	Save(ctx context.Context) error

	// Config return the config of cluster
	Config(ctx context.Context) (*internalversion.KwokctlConfiguration, error)

	// Install the cluster
	Install(ctx context.Context) error

	// Uninstall the cluster
	Uninstall(ctx context.Context) error

	// Up start the cluster
	Up(ctx context.Context) error

	// Down stop the cluster
	Down(ctx context.Context) error

	// Start a cluster
	Start(ctx context.Context) error

	// Stop a cluster
	Stop(ctx context.Context) error

	// StartComponent start cluster component
	StartComponent(ctx context.Context, name string) error

	// StopComponent stop cluster component
	StopComponent(ctx context.Context, name string) error

	// GetComponent return the component if it exists
	GetComponent(ctx context.Context, name string) (internalversion.Component, error)

	// ListComponents list the components of cluster
	ListComponents(ctx context.Context) ([]internalversion.Component, error)

	// InspectComponent inspect the component
	InspectComponent(ctx context.Context, name string) (ComponentStatus, error)

	// PortForward expose the port of the component
	PortForward(ctx context.Context, name string, portOrName string, hostPort uint32) (cancel func(), retErr error)

	// Ready check the cluster is ready
	Ready(ctx context.Context) (bool, error)

	// WaitReady wait the cluster is ready
	WaitReady(ctx context.Context, timeout time.Duration) error

	// AddContext add the context of cluster to kubeconfig
	AddContext(ctx context.Context, kubeconfigPath string) error

	// RemoveContext remove the context of cluster from kubeconfig
	RemoveContext(ctx context.Context, kubeconfigPath string) error

	// Kubectl command
	Kubectl(ctx context.Context, args ...string) error

	// KubectlInCluster command in cluster
	KubectlInCluster(ctx context.Context, args ...string) error

	// EtcdctlInCluster command in cluster
	EtcdctlInCluster(ctx context.Context, args ...string) error

	// Logs logs of a component
	Logs(ctx context.Context, name string, out io.Writer) error

	// LogsFollow follow logs of a component with follow
	LogsFollow(ctx context.Context, name string, out io.Writer) error

	// CollectLogs will populate dir with cluster logs and other debug files
	CollectLogs(ctx context.Context, dir string) error

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

	// SnapshotSaveWithYAML save the snapshot of cluster
	SnapshotSaveWithYAML(ctx context.Context, path string, conf SnapshotSaveWithYAMLConfig) error

	// SnapshotRestoreWithYAML restore the snapshot of cluster
	SnapshotRestoreWithYAML(ctx context.Context, path string, conf SnapshotRestoreWithYAMLConfig) error

	// GetWorkdirPath get the workdir path of cluster
	GetWorkdirPath(name string) string

	// InitCRDs init the crds of cluster
	InitCRDs(ctx context.Context) error

	// InitCRs init the crs of cluster
	InitCRs(ctx context.Context) error

	// IsDryRun returns true if the runtime is in dry-run mode
	IsDryRun() bool

	// GetClientset returns the clientset of cluster
	GetClientset(ctx context.Context) (client.Clientset, error)

	// Kectl executes kectl commands on the cluster from the host
	Kectl(ctx context.Context, args ...string) error

	// KectlInCluster executes kectl commands from within the cluster
	KectlInCluster(ctx context.Context, args ...string) error
}

// SnapshotSaveWithYAMLConfig contains configuration for saving a snapshot with YAML files
type SnapshotSaveWithYAMLConfig struct {
	// Filters specifies which resources to include in the snapshot
	Filters []string
}

// SnapshotRestoreWithYAMLConfig contains configuration for restoring a snapshot with YAML files
type SnapshotRestoreWithYAMLConfig struct {
	// Filters specifies which resources to restore from the snapshot
	Filters []string
}

// ComponentStatus represents the status of a cluster component
type ComponentStatus uint64

const (
	// ComponentStatusUnknown indicates the component status is unknown
	ComponentStatusUnknown ComponentStatus = iota
	// ComponentStatusStopped indicates the component is stopped
	ComponentStatusStopped
	// ComponentStatusRunning indicates the component is running
	ComponentStatusRunning
	// ComponentStatusReady indicates the component is ready to serve requests
	ComponentStatusReady
)
