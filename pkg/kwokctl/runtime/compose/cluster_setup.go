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

package compose

import (
	"context"
	"fmt"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/sets"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func (c *Cluster) setup(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options
	if !file.Exists(env.pkiPath) {
		var sans []string
		if !c.isHostNetwork {
			sans = []string{
				c.Name() + "-kube-apiserver",
				c.Name() + "-kwok-controller",
			}
		}
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
		err = c.MkdirAll(env.pkiPath)
		if err != nil {
			return fmt.Errorf("failed to create pki dir: %w", err)
		}
		err = c.GeneratePki(env.pkiPath, sans...)
		if err != nil {
			return fmt.Errorf("failed to generate pki: %w", err)
		}
	}

	if conf.KubeAuditPolicy != "" {
		err := c.MkdirAll(c.GetWorkdirPath("logs"))
		if err != nil {
			return err
		}

		err = c.CreateFile(env.auditLogPath)
		if err != nil {
			return err
		}

		err = c.CopyFile(conf.KubeAuditPolicy, env.auditPolicyPath)
		if err != nil {
			return err
		}
	}

	err := c.MkdirAll(env.etcdDataPath)
	if err != nil {
		return fmt.Errorf("failed to mkdir etcd data path: %w", err)
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

	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

	if c.isHostNetwork {
		err = c.setupPorts(ctx,
			env.usedPorts,
			&env.kwokctlConfig.Options.EtcdPeerPort,
			&env.kwokctlConfig.Options.EtcdPort,
			&env.kwokctlConfig.Options.KubeApiserverPort,
			&env.kwokctlConfig.Options.KwokControllerPort,
		)
		if err != nil {
			return err
		}

		if env.kwokctlConfig.Options.JaegerPort != 0 {
			err = c.setupPorts(ctx,
				env.usedPorts,
				&env.kwokctlConfig.Options.JaegerOtlpGrpcPort,
			)
			if err != nil {
				return err
			}
		}
	} else {
		err = c.setupPorts(ctx,
			env.usedPorts,
			&env.kwokctlConfig.Options.KubeApiserverPort,
		)
		if err != nil {
			return err
		}
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

	err = c.addSchedulerPlugins(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKueue(ctx, env)
	if err != nil {
		return err
	}

	err = c.addKueueviz(ctx, env)
	if err != nil {
		return err
	}

	err = c.addJobSet(ctx, env)
	if err != nil {
		return err
	}

	err = c.addLWS(ctx, env)
	if err != nil {
		return err
	}

	err = c.addDescheduler(ctx, env)
	if err != nil {
		return err
	}

	err = c.addNodeReadinessController(ctx, env)
	if err != nil {
		return err
	}

	err = c.setupPrometheusConfig(ctx, env)
	if err != nil {
		return err
	}

	err = c.finishInstall(ctx, env)
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

func (c *Cluster) finishInstall(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	for i := range env.kwokctlConfig.Components {
		runtime.ApplyComponentPatches(ctx, &env.kwokctlConfig.Components[i], env.kwokctlConfig.ComponentsPatches)
	}

	// Setup kubeconfig
	kubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
		ProjectName:  c.Name(),
		SecurePort:   conf.SecurePort,
		Address:      env.scheme + "://" + utilsnet.LocalAddress + ":" + format.String(conf.KubeApiserverPort),
		CACrtPath:    env.caCertPath,
		AdminCrtPath: env.adminCertPath,
		AdminKeyPath: env.adminKeyPath,
	}))
	if err != nil {
		return err
	}

	var inClusterKubeconfigData []byte
	if c.isHostNetwork {
		inClusterKubeconfigData, err = kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
			ProjectName:  c.Name(),
			SecurePort:   conf.SecurePort,
			Address:      env.scheme + "://" + utilsnet.LocalAddress + ":" + format.String(conf.KubeApiserverPort),
			CACrtPath:    env.inClusterCaCertPath,
			AdminCrtPath: env.inClusterAdminCertPath,
			AdminKeyPath: env.inClusterAdminKeyPath,
		}))
	} else {
		inClusterKubeconfigData, err = kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
			ProjectName:  c.Name(),
			SecurePort:   conf.SecurePort,
			Address:      env.scheme + "://" + c.Name() + "-kube-apiserver:" + format.String(env.inClusterPort),
			CACrtPath:    env.inClusterCaCertPath,
			AdminCrtPath: env.inClusterAdminCertPath,
			AdminKeyPath: env.inClusterAdminKeyPath,
		}))
	}
	if err != nil {
		return err
	}

	// Save config
	err = c.WriteFile(env.kubeconfigPath, kubeconfigData)
	if err != nil {
		return err
	}

	err = c.WriteFile(env.inClusterOnHostKubeconfigPath, inClusterKubeconfigData)
	if err != nil {
		return err
	}

	err = c.SetConfig(ctx, env.kwokctlConfig)
	if err != nil {
		return err
	}
	err = c.Save(ctx)
	if err != nil {
		return err
	}

	if !c.isHostNetwork {
		err = c.createNetwork(ctx)
		if err != nil {
			return err
		}
	}

	err = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err = c.createComponents(ctx)
		return err == nil, err
	},
		wait.WithContinueOnError(5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}

	return nil
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	err := wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		err := c.deleteComponents(ctx)
		return err == nil, err
	},
		wait.WithContinueOnError(5),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}

	if !c.isHostNetwork {
		err = c.deleteNetwork(ctx)
		if err != nil {
			return err
		}
	}

	err = c.Cluster.Uninstall(ctx)
	if err != nil {
		return err
	}
	return nil
}
