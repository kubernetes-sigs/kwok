/*
Copyright 2023 The Kubernetes Authors.

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
	"bytes"
	"context"
	"os"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
)

// SnapshotSaveWithYAML save the snapshot of cluster
func (c *Cluster) SnapshotSaveWithYAML(ctx context.Context, path string, filters []string, pageSize int64, pageBufferSize int32) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	kubeconfigPath := c.GetWorkdirPath(InHostKubeconfigName)
	// In most cases, the user should have full privileges on the clusters created by kwokctl,
	// so no need to expose impersonation args to "snapshot save" command.
	return snapshot.Save(ctx, kubeconfigPath, file, filters, rest.ImpersonationConfig{}, pageSize, pageBufferSize)
}

// SnapshotRestoreWithYAML restore the snapshot of cluster
func (c *Cluster) SnapshotRestoreWithYAML(ctx context.Context, path string, filters []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	kubeconfigPath := c.GetWorkdirPath(InHostKubeconfigName)
	return snapshot.Load(ctx, kubeconfigPath, bytes.NewBuffer(data), filters)
}
