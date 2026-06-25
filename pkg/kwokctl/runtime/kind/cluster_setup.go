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

package kind

import (
	"context"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/sets"
)

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	err := c.Cluster.Install(ctx)
	if err != nil {
		return err
	}

	env, err := c.env(ctx)
	if err != nil {
		return err
	}

	err = c.preInstall(ctx, env)
	if err != nil {
		return err
	}

	// This is not necessary when creating a cluster use kind, but in Linux the cluster is created as root,
	// and the files here may not have permissions when deleted, so we create them first.
	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&env.kwokctlConfig.Options.KubeApiserverPort,
	)
	if err != nil {
		return err
	}

	err = c.addKind(ctx, env)
	if err != nil {
		return err
	}

	err = c.addEtcd(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeApiserver(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubectlProxy(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeControllerManager(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKubeScheduler(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKwokController(ctx, env)
	if err != nil {
		return err
	}

	err = c.addMetricsServer(ctx, env)
	if err != nil {
		return err
	}

	err = c.addPrometheus(ctx, env)
	if err != nil {
		return err
	}

	err = c.addJaeger(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPrometheusConfig(ctx, env)
	if err != nil {
		return err
	}

	err = c.CheckComponentIssues(ctx, env.kwokctlConfig.Components)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) preInstall(_ context.Context, env *env) error {
	for i, patch := range env.kwokctlConfig.ComponentsPatches {
		if len(patch.ExtraVolumes) == 0 {
			continue
		}
		volumes, err := runtime.ExpandVolumesHostPaths(patch.ExtraVolumes)
		if err != nil {
			return fmt.Errorf("failed to expand host volumes for %q component: %w", patch.Name, err)
		}

		env.kwokctlConfig.ComponentsPatches[i].ExtraVolumes = volumes
	}
	return nil
}

func (c *Cluster) setup(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	pkiPath := c.GetWorkdirPath(runtime.PkiName)
	if !file.Exists(pkiPath) {
		sans := []string{}
		ips, err := utilsnet.GetAllIPs()
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Warn("failed to get all ips",
				"err", err,
			)
		} else {
			sans = append(sans, ips...)
		}
		if len(conf.KubeApiserverCertSANs) != 0 {
			sans = append(sans, conf.KubeApiserverCertSANs...)
		}
		err = c.MkdirAll(pkiPath)
		if err != nil {
			return fmt.Errorf("failed to create pki dir: %w", err)
		}
		err = c.GeneratePki(pkiPath, sans...)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	pkiEtcd := filepath.Join(pkiPath, "etcd")
	err := c.MkdirAll(pkiEtcd)
	if err != nil {
		return fmt.Errorf("failed to create pki dir: %w", err)
	}
	return nil
}

func (c *Cluster) setupPorts(ctx context.Context, used sets.Sets[uint32], ports ...*uint32) error {
	for _, port := range ports {
		if port != nil && *port == 0 {
			p, err := utilsnet.GetUnusedPort(ctx, used)
			if err != nil {
				return err
			}
			*port = p
		}
	}
	return nil
}
