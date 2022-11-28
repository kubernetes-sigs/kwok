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

package kind

import (
	"context"
	"os"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
)

// SnapshotSave save the snapshot of cluster
func (c *Cluster) SnapshotSave(ctx context.Context, path string) error {
	etcdContainerName, err := c.getComponentName("etcd")
	if err != nil {
		return err
	}
	kindName, err := c.getClusterName()
	if err != nil {
		return err
	}

	// Save to /var/lib/etcd/snapshot.db on container of Kind use Kubectl
	// Why this path, etcd is just the volume /var/lib/etcd/ in container of Kind.
	tmpFile := "/var/lib/etcd/snapshot.db"
	err = c.KubectlInCluster(ctx, utils.IOStreams{}, "exec", "-i", "-n", "kube-system", etcdContainerName, "--", "etcdctl", "snapshot", "save", tmpFile, "--endpoints=127.0.0.1:2379", "--cert=/etc/kubernetes/pki/etcd/server.crt", "--key=/etc/kubernetes/pki/etcd/server.key", "--cacert=/etc/kubernetes/pki/etcd/ca.crt")
	if err != nil {
		return err
	}
	defer func() {
		err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "exec", "-i", kindName, "rm", "-f", tmpFile)
		if err != nil {
			c.Logger().Error("Failed to clean snapshot", err)
		}
	}()

	// Copy to host path from container of Kind use Docker
	// Etcd image does not have `tar`, can't use `kubectl cp`, so we use `docker cp` instead
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "cp", kindName+":"+tmpFile, path)
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
	kindName, err := c.getClusterName()
	if err != nil {
		return err
	}

	bin := utils.PathJoin(conf.Workdir, "bin")
	etcdctlPath := utils.PathJoin(bin, "etcdctl"+vars.BinSuffix)

	err = utils.DownloadWithCacheAndExtract(ctx, conf.CacheDir, conf.EtcdBinaryTar, etcdctlPath, "etcdctl"+vars.BinSuffix, 0755, conf.QuietPull, true)
	if err != nil {
		return err
	}

	err = c.Stop(ctx, "etcd")
	if err != nil {
		c.Logger().Error("Failed to stop etcd", err)
	}
	defer func() {
		err = c.Start(ctx, "etcd")
		if err != nil {
			c.Logger().Error("Failed to start etcd", err)
		}
	}()

	// Restore snapshot to host temporary directory
	etcdDataTmp := utils.PathJoin(conf.Workdir, "etcd")
	err = utils.Exec(ctx, "", utils.IOStreams{}, etcdctlPath, "snapshot", "restore", path, "--data-dir", etcdDataTmp)
	if err != nil {
		return err
	}
	defer func() {
		os.RemoveAll(etcdDataTmp)
	}()

	// Copy to kind container from host temporary directory
	err = utils.Exec(ctx, "", utils.IOStreams{}, "docker", "cp", etcdDataTmp, kindName+":/var/lib/")
	if err != nil {
		return err
	}

	return nil
}
