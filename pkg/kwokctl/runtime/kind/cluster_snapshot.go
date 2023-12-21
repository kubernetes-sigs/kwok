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
	"crypto/tls"

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
	kindName := c.getClusterName()

	logger := log.FromContext(ctx)

	// Save to /var/lib/etcd/snapshot.db on container of Kind use Kubectl
	// Why this path, etcd is just the volume /var/lib/etcd/ in container of Kind.
	tmpFile := "/var/lib/etcd/snapshot.db"
	err := c.EtcdctlInCluster(ctx, "snapshot", "save", tmpFile)
	if err != nil {
		return err
	}
	defer func() {
		err = c.Exec(ctx, c.runtime, "exec", "-i", kindName, "rm", "-f", tmpFile)
		if err != nil {
			logger.Error("Failed to clean snapshot", err)
		}
	}()

	// Copy to host path from container of Kind use Docker
	// Etcd image does not have `tar`, can't use `kubectl cp`, so we use `docker cp` instead
	err = c.Exec(ctx, c.runtime, "cp", kindName+":"+tmpFile, path)
	if err != nil {
		return err
	}

	return nil
}

// SnapshotRestore restore the snapshot of cluster
func (c *Cluster) SnapshotRestore(ctx context.Context, path string) error {
	logger := log.FromContext(ctx)
	clusterName := c.getClusterName()

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
		for _, component := range components {
			err := c.StartComponent(ctx, component)
			if err != nil {
				logger.Error("Failed to start", err, "component", component)
			}
		}

		err := c.Stop(ctx)
		if err != nil {
			logger.Error("Failed to stop", err)
		}
		err = c.Start(ctx)
		if err != nil {
			logger.Error("Failed to start", err)
		}
	}()

	// Restore snapshot to host temporary directory
	etcdDataTmp := c.GetWorkdirPath(consts.ComponentEtcd)
	err := c.Etcdctl(ctx, "snapshot", "restore", path, "--data-dir", etcdDataTmp)
	if err != nil {
		return err
	}
	defer func() {
		err = c.RemoveAll(etcdDataTmp)
		if err != nil {
			logger.Error("Failed to clear etcd temporary data", err)
		}
	}()

	// Copy to kind container from host temporary directory
	err = c.Exec(ctx, c.runtime, "cp", etcdDataTmp, clusterName+":/var/lib/")
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
	err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := c.StopComponent(ctx, consts.ComponentKubeControllerManager)
		return err == nil, err
	})
	if err != nil {
		logger.Error("Failed to stop kube-controller-manager", err)
	}
	defer func() {
		err = c.StartComponent(ctx, consts.ComponentKubeControllerManager)
		if err != nil {
			logger.Error("Failed to start kube-controller-manager", err)
		}
	}()

	err = c.Cluster.SnapshotRestoreWithYAML(ctx, path, conf)
	if err != nil {
		return err
	}
	return nil
}

// GetEtcdClient returns the etcd client of cluster
func (c *Cluster) GetEtcdClient(ctx context.Context) (etcd.Client, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}
	conf := &config.Options

	certFile := c.GetWorkdirPath("pki/etcd/server.crt")
	keyFile := c.GetWorkdirPath("pki/etcd/server.key")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return etcd.NewClient(etcd.ClientConfig{
		Endpoints: []string{"https://" + net.LocalAddress + ":" + format.String(conf.EtcdPort)},
		TLS: &tls.Config{
			Certificates: []tls.Certificate{cert},
			//nolint:gosec
			InsecureSkipVerify: true,
		},
	})
}
