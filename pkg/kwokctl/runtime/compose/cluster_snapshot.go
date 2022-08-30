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
	"os"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
)

// SnapshotSave save the snapshot of cluster
func (c *Cluster) SnapshotSave(ctx context.Context, path string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}
	etcdContainerName := conf.Name + "-etcd"

	// Save to /snapshot.db on container
	tmpFile := "/snapshot.db"
	err = utils.Exec(ctx, "", utils.IOStreams{}, conf.Runtime, "exec", "-i", etcdContainerName, "etcdctl", "snapshot", "save", tmpFile)
	if err != nil {
		return err
	}

	// Copy to host path from container
	err = utils.Exec(ctx, "", utils.IOStreams{}, conf.Runtime, "cp", etcdContainerName+":"+tmpFile, path)
	if err != nil {
		return err
	}
	return nil
}

// SnapshotRestore restore the snapshot of cluster
func (c *Cluster) SnapshotRestore(ctx context.Context, path string) error {
	conf, err := c.Config()
	if err != nil {
		return err
	}

	etcdContainerName := conf.Name + "-etcd"

	bin := utils.PathJoin(conf.Workdir, "bin")
	etcdctlPath := utils.PathJoin(bin, "etcdctl"+vars.BinSuffix)

	err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdctlPath, "etcdctl"+vars.BinSuffix, 0755, conf.QuietPull, true)
	if err != nil {
		return err
	}

	err = c.Stop(ctx, "etcd")
	if err != nil {
		c.Logger().Printf("Failed to stop etcd: %v", err)
	}
	defer func() {
		err = c.Start(ctx, "etcd")
		if err != nil {
			c.Logger().Printf("Failed to start etcd: %v", err)
		}
	}()

	// Restore snapshot to host temporary directory
	etcdDataTmp := utils.PathJoin(conf.Workdir, "etcd-data")
	err = utils.Exec(ctx, "", utils.IOStreams{}, etcdctlPath, "snapshot", "restore", path, "--data-dir", etcdDataTmp)
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(etcdDataTmp)
	}()

	// Copy to container from host temporary directory
	err = utils.Exec(ctx, "", utils.IOStreams{}, conf.Runtime, "cp", etcdDataTmp, etcdContainerName+":/")
	if err != nil {
		return err
	}
	return nil
}
