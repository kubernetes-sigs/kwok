/*
Copyright 2026 The Kubernetes Authors.

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
	"os/exec"

	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// KubectlPath returns the path to the kubectl binary. It first tries to find kubectl in the system PATH.
// If not found, it will download and install the kubectl binary using the configured version.
// Returns the path to the kubectl binary or an error if it cannot be found or installed.
func (c *Cluster) KubectlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		kubectlPath, err = c.EnsureBinary(ctx, "kubectl", conf.KubectlBinary)
		if err != nil {
			return "", err
		}
	}
	return kubectlPath, nil
}

// Kubectl runs kubectl.
func (c *Cluster) Kubectl(ctx context.Context, args ...string) error {
	kubectlPath, err := c.KubectlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, kubectlPath, args...)
}

// KubectlInCluster runs kubectl in the cluster.
func (c *Cluster) KubectlInCluster(ctx context.Context, args ...string) error {
	kubectlPath, err := c.KubectlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, kubectlPath, append([]string{"--kubeconfig", c.GetWorkdirPath(InHostKubeconfigName)}, args...)...)
}

func (c *Cluster) etcdctlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options
	etcdctlPath, err := c.EnsureBinary(ctx, "etcdctl", conf.EtcdctlBinary)
	if err != nil {
		return "", err
	}
	return etcdctlPath, nil
}

func (c *Cluster) etcdutlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options

	etcdutlPath, err := c.EnsureBinary(ctx, "etcdutl", conf.EtcdutlBinary)
	if err != nil {
		return "", err
	}
	return etcdutlPath, nil
}

// Etcdctl runs etcdctl.
func (c *Cluster) Etcdctl(ctx context.Context, args ...string) error {
	etcdctlPath, err := c.etcdctlPath(ctx)
	if err != nil {
		return err
	}

	// If using versions earlier than v3.4, set `ETCDCTL_API=3` to use v3 API.
	ctx = utilsexec.WithEnv(ctx, []string{"ETCDCTL_API=3"})
	return c.Exec(ctx, etcdctlPath, args...)
}

// Etcdutl runs etcdutl.
func (c *Cluster) Etcdutl(ctx context.Context, args ...string) error {
	etcdutlPath, err := c.etcdutlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, etcdutlPath, args...)
}

func (c *Cluster) kectlPath(ctx context.Context) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := &config.Options
	kectlPath, err := c.EnsureBinary(ctx, "kectl", conf.KectlBinary)
	if err != nil {
		return "", err
	}
	return kectlPath, nil
}

// Kectl runs kectl.
func (c *Cluster) Kectl(ctx context.Context, args ...string) error {
	kectlPath, err := c.kectlPath(ctx)
	if err != nil {
		return err
	}

	return c.Exec(ctx, kectlPath, args...)
}

// Snapshot takes a snapshot of the cluster with the given arguments.
// The arguments are passed to etcdctl or etcdutl, which depends on the etcd version.
func (c *Cluster) Snapshot(ctx context.Context, args ...string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	etcdVersion, err := version.ParseVersion(conf.EtcdVersion)
	if err != nil {
		return err
	}

	// etcdctl is remove snapshot restore in v3.6,
	// so use etcdutl for v3.6 and later, and use etcdctl for earlier versions.
	if etcdVersion.LT(version.NewVersion(3, 6, 0)) {
		return c.Etcdctl(ctx, args...)
	}

	return c.Etcdutl(ctx, args...)
}
