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

package binary

import (
	"context"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// SnapshotSave save the snapshot of cluster
func (c *Cluster) SnapshotSave(ctx context.Context, path string) error {
	err := c.EtcdctlInCluster(ctx, "snapshot", "save", path)
	if err != nil {
		return err
	}

	return nil
}

// SnapshotRestore restore the snapshot of cluster
func (c *Cluster) SnapshotRestore(ctx context.Context, path string) error {
	logger := log.FromContext(ctx)

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

	etcdDataTmp := c.GetWorkdirPath("etcd-data")
	err := c.RemoveAll(etcdDataTmp)
	if err != nil {
		return err
	}

	err = c.EtcdctlInCluster(ctx, "snapshot", "restore", path, "--data-dir", etcdDataTmp)
	if err != nil {
		return err
	}

	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	err = c.RemoveAll(etcdDataPath)
	if err != nil {
		return err
	}
	err = c.RenameFile(etcdDataTmp, etcdDataPath)
	if err != nil {
		return err
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
		err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
			err := c.StopComponent(ctx, component)
			if err != nil {
				return false, err
			}
			component, err := c.GetComponent(ctx, component)
			if err != nil {
				return false, err
			}
			ready := c.isRunning(ctx, component)
			return !ready, nil
		})
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

// GetEtcdClient returns the etcd client of cluster
func (c *Cluster) GetEtcdClient(ctx context.Context) (etcd.Client, func(), error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, nil, err
	}
	conf := &config.Options

	cli, err := etcd.NewClient(etcd.ClientConfig{
		Endpoints: []string{"http://" + net.LocalAddress + ":" + format.String(conf.EtcdPort)},
	})
	if err != nil {
		return nil, nil, err
	}

	return cli, func() {}, nil
}
