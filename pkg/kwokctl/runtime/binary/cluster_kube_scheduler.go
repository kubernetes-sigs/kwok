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
	"slices"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
)

func (c *Cluster) addKubeScheduler(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeScheduler) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kube-scheduler
	kubeSchedulerBinary := conf.KubeSchedulerBinary
	if slices.Contains(env.components, consts.ComponentSchedulerPlugins) {
		kubeSchedulerBinary = conf.SchedulerPluginsSchedulerBinary
	}
	kubeSchedulerPath, err := c.EnsureBinary(ctx, consts.ComponentKubeScheduler, kubeSchedulerBinary)
	if err != nil {
		return err
	}

	schedulerConfigPath := ""
	if conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.inClusterKubeconfigPath)
		if err != nil {
			return err
		}
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&conf.KubeSchedulerPort,
	)
	if err != nil {
		return err
	}

	kubeSchedulerVersion, err := c.ParseVersionFromBinary(ctx, kubeSchedulerPath)
	if err != nil {
		return err
	}

	kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Binary:           kubeSchedulerPath,
		Version:          kubeSchedulerVersion,
		BindAddress:      conf.BindAddress,
		Port:             conf.KubeSchedulerPort,
		SecurePort:       conf.SecurePort,
		CaCertPath:       env.caCertPath,
		AdminCertPath:    env.adminCertPath,
		AdminKeyPath:     env.adminKeyPath,
		ConfigPath:       schedulerConfigPath,
		KubeconfigPath:   env.inClusterKubeconfigPath,
		KubeFeatureGates: conf.KubeFeatureGates,
		Verbosity:        env.verbosity,
		DisableQPSLimits: conf.DisableQPSLimits,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeSchedulerComponent)
	return nil
}
