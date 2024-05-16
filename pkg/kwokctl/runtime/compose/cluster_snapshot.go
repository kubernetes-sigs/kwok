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

package compose

import (
	"context"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
)

// SnapshotSave save the snapshot of cluster
func (c *Cluster) SnapshotSave(ctx context.Context, path string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	// Save to /snapshot.db on container
	tmpFile := "/snapshot.db"
	err = c.EtcdctlInCluster(ctx, "snapshot", "save", tmpFile)
	if err != nil {
		return err
	}

	etcdContainerName := c.Name() + "-etcd"
	// Copy to host path from container
	err = c.Exec(ctx, conf.Runtime, "cp", etcdContainerName+":"+tmpFile, path)
	if err != nil {
		return err
	}
	return nil
}

// SnapshotRestore restore the snapshot of cluster
func (c *Cluster) SnapshotRestore(ctx context.Context, path string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	logger := log.FromContext(ctx)
	// Restore snapshot to host temporary directory
	etcdDataTmp := c.GetWorkdirPath("etcd-data")
	err = c.Etcdctl(ctx, "snapshot", "restore", path, "--data-dir", etcdDataTmp)
	if err != nil {
		return err
	}
	defer func() {
		err = c.RemoveAll(etcdDataTmp)
		if err != nil {
			logger.Error("Failed to clear etcd temporary data", err)
		}
	}()

	etcdContainerName := c.Name() + "-etcd"
	if !c.isNerdctl {
		// Restart etcd and kube-apiserver
		components := []string{
			consts.ComponentEtcd,
			consts.ComponentKubeApiserver,
		}
		for _, component := range components {
			err := c.StopComponent(ctx, component)
			if err != nil {
				logger.Error("Failed to stop", err, "component", component)
			}
		}
		defer func() {
			for _, component := range components {
				err := c.StartComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to start", err, "component", component)
				}
			}

			components := []string{
				consts.ComponentKwokController,
				consts.ComponentKubeControllerManager,
				consts.ComponentKubeScheduler,
			}
			for _, component := range components {
				err := c.StopComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to stop", err, "component", component)
				}
				err = c.StartComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to start", err, "component", component)
				}
			}
		}()

		// Copy to container from host temporary directory
		err = c.Exec(ctx, conf.Runtime, "cp", etcdDataTmp, etcdContainerName+":/")
		if err != nil {
			return err
		}
	} else {
		// TODO: remove this when `nerdctl cp` supports work on stopped containers
		// https://github.com/containerd/nerdctl/issues/1812

		// Stop the kube-apiserver container to avoid data modification by etcd during restore.
		err = c.StopComponent(ctx, consts.ComponentKubeApiserver)
		if err != nil {
			logger.Error("Failed to stop kube-apiserver", err)
		}
		defer func() {
			err = c.StartComponent(ctx, consts.ComponentKubeApiserver)
			if err != nil {
				logger.Error("Failed to start kube-apiserver", err)
			}
		}()

		// Copy to container from host temporary directory
		err = c.Exec(ctx, conf.Runtime, "cp", etcdDataTmp, etcdContainerName+":/")
		if err != nil {
			return err
		}

		// Restart etcd and kube-apiserver
		components := []string{
			consts.ComponentEtcd,
		}
		for _, component := range components {
			err := c.StopComponent(ctx, component)
			if err != nil {
				logger.Error("Failed to stop", err, "component", component)
			}
		}
		defer func() {
			components := []string{
				consts.ComponentEtcd,
				consts.ComponentKubeApiserver,
			}
			for _, component := range components {
				err := c.StartComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to start", err, "component", component)
				}
			}

			components = []string{
				consts.ComponentKwokController,
				consts.ComponentKubeControllerManager,
				consts.ComponentKubeScheduler,
			}
			for _, component := range components {
				err := c.StopComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to stop", err, "component", component)
				}
				err = c.StartComponent(ctx, component)
				if err != nil {
					logger.Error("Failed to start", err, "component", component)
				}
			}
		}()
	}

	return nil
}

// SnapshotSaveWithYAML save the snapshot of cluster
func (c *Cluster) SnapshotSaveWithYAML(ctx context.Context, path string, conf runtime.SnapshotSaveWithYAMLConfig) error {
	err := c.Cluster.SnapshotSaveWithYAML(ctx, path, conf)
	if err != nil {
		return err
	}
	return nil
}

// SnapshotRestoreWithYAML restore the snapshot of cluster
func (c *Cluster) SnapshotRestoreWithYAML(ctx context.Context, path string, conf runtime.SnapshotRestoreWithYAMLConfig) error {
	logger := log.FromContext(ctx)
	components := []string{
		consts.ComponentKubeScheduler,
		consts.ComponentKubeControllerManager,
		consts.ComponentKwokController,
	}
	for _, component := range components {
		err := c.StopComponent(ctx, component)
		if err != nil {
			logger.Error("Failed to stop", err, "component", component)
		}
	}
	defer func() {
		for _, component := range components {
			err := c.StartComponent(ctx, component)
			if err != nil {
				logger.Error("Failed to start", err, "component", component)
			}
		}
	}()

	err := c.Cluster.SnapshotRestoreWithYAML(ctx, path, conf)
	if err != nil {
		return err
	}
	return nil
}
