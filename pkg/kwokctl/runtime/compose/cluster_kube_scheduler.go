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
	"slices"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addKubeScheduler(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeScheduler) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kube-scheduler
	schedulerConfigPath := ""
	if conf.KubeSchedulerConfig != "" {
		schedulerConfigPath = c.GetWorkdirPath(runtime.SchedulerConfigName)
		err = c.CopySchedulerConfig(conf.KubeSchedulerConfig, schedulerConfigPath, env.inClusterKubeconfig)
		if err != nil {
			return err
		}
	}

	kubeSchedulerImage := conf.KubeSchedulerImage
	if slices.Contains(env.components, consts.ComponentSchedulerPlugins) {
		kubeSchedulerImage = conf.SchedulerPluginsSchedulerImage
	}

	err = c.EnsureImage(ctx, c.runtime, kubeSchedulerImage)
	if err != nil {
		return err
	}

	kubeSchedulerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, kubeSchedulerImage, consts.ComponentKubeScheduler)
	if err != nil {
		return err
	}

	kubeSchedulerComponent, err := components.BuildKubeSchedulerComponent(components.BuildKubeSchedulerComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Image:            kubeSchedulerImage,
		Version:          kubeSchedulerVersion,
		BindAddress:      utilsnet.PublicAddress,
		Port:             conf.KubeSchedulerPort,
		SecurePort:       conf.SecurePort,
		CaCertPath:       env.caCertPath,
		AdminCertPath:    env.adminCertPath,
		AdminKeyPath:     env.adminKeyPath,
		ConfigPath:       schedulerConfigPath,
		KubeconfigPath:   env.inClusterOnHostKubeconfigPath,
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
