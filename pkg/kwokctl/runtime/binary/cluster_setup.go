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

package binary

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
)

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

	if conf.KubeAuditPolicy != "" {
		auditLogPath := c.GetLogPath(runtime.AuditLogName)
		err := c.CreateFile(auditLogPath)
		if err != nil {
			return err
		}

		auditPolicyPath := c.GetWorkdirPath(runtime.AuditPolicyName)
		err = c.CopyFile(conf.KubeAuditPolicy, auditPolicyPath)
		if err != nil {
			return err
		}
	}

	etcdDataPath := c.GetWorkdirPath(runtime.EtcdDataDirName)
	err := c.MkdirAll(etcdDataPath)
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

func (c *Cluster) checkRunInCluster(ctx context.Context) {
	if !file.Exists("/var/run/secrets/kubernetes.io/serviceaccount/token") {
		return
	}

	logger := log.FromContext(ctx)
	logger.Warn("cluster may not work correctly and need to be workaround." +
		"see https://kwok.sigs.k8s.io/docs/user/all-in-one-image/#use-in-a-pod")
}

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	c.checkRunInCluster(ctx)

	err := c.Cluster.Install(ctx)
	if err != nil {
		return err
	}

	dirs := []string{
		"pids",
		"logs",
	}

	for _, dir := range dirs {
		err = c.MkdirAll(c.GetWorkdirPath(dir))
		if err != nil {
			return err
		}
	}

	env, err := c.env(ctx)
	if err != nil {
		return err
	}

	err = c.setup(ctx, env)
	if err != nil {
		return err
	}

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

func (c *Cluster) finishInstall(ctx context.Context, env *env) error {
	conf := &env.kwokctlConfig.Options

	for i := range env.kwokctlConfig.Components {
		runtime.ApplyComponentPatches(ctx, &env.kwokctlConfig.Components[i], env.kwokctlConfig.ComponentsPatches)
	}

	// Setup kubeconfig
	inClusterKubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
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
	err = c.WriteFile(env.inClusterKubeconfigPath, inClusterKubeconfigData)
	if err != nil {
		return err
	}

	if conf.KubeApiserverInsecurePort != 0 {
		kubeconfigData, err := kubeconfig.EncodeKubeconfig(kubeconfig.BuildKubeconfig(kubeconfig.BuildKubeconfigConfig{
			ProjectName: c.Name(),
			SecurePort:  false,
			Address:     "http://" + utilsnet.LocalAddress + ":" + format.String(conf.KubeApiserverInsecurePort),
		}))
		if err != nil {
			return err
		}
		err = c.WriteFile(env.kubeconfigPath, kubeconfigData)
		if err != nil {
			return err
		}
	}

	// Save config
	err = c.SetConfig(ctx, env.kwokctlConfig)
	if err != nil {
		return err
	}
	err = c.Save(ctx)
	if err != nil {
		return err
	}

	return nil
}
