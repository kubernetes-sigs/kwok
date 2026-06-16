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
)

func (c *Cluster) addKubeControllerManager(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeControllerManager) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kube-controller-manager
	kubeControllerManagerPath, err := c.EnsureBinary(ctx, consts.ComponentKubeControllerManager, conf.KubeControllerManagerBinary)
	if err != nil {
		return err
	}

	err = c.setupPorts(ctx,
		env.usedPorts,
		&conf.KubeControllerManagerPort,
	)
	if err != nil {
		return err
	}

	kubeControllerManagerVersion, err := c.ParseVersionFromBinary(ctx, kubeControllerManagerPath)
	if err != nil {
		return err
	}

	kubeControllerManagerComponent, err := components.BuildKubeControllerManagerComponent(components.BuildKubeControllerManagerComponentConfig{
		Runtime:                            conf.Runtime,
		ProjectName:                        c.Name(),
		Workdir:                            env.workdir,
		Binary:                             kubeControllerManagerPath,
		Version:                            kubeControllerManagerVersion,
		BindAddress:                        conf.BindAddress,
		Port:                               conf.KubeControllerManagerPort,
		SecurePort:                         conf.SecurePort,
		CaCertPath:                         env.caCertPath,
		AdminCertPath:                      env.adminCertPath,
		AdminKeyPath:                       env.adminKeyPath,
		KubeAuthorization:                  conf.KubeAuthorization,
		KubeconfigPath:                     env.inClusterKubeconfigPath,
		KubeFeatureGates:                   conf.KubeFeatureGates,
		NodeMonitorPeriodMilliseconds:      conf.KubeControllerManagerNodeMonitorPeriodMilliseconds,
		NodeMonitorGracePeriodMilliseconds: conf.KubeControllerManagerNodeMonitorGracePeriodMilliseconds,
		Verbosity:                          env.verbosity,
		DisableQPSLimits:                   conf.DisableQPSLimits,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeControllerManagerComponent)
	return nil
}
