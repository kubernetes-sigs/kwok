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

func (c *Cluster) addKubectlProxy(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeApiserverInsecureProxy) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kubectl
	kubectlPath, err := c.KubectlPath(ctx)
	if err != nil {
		return err
	}

	kubectlProxyComponent, err := components.BuildKubectlProxyComponent(components.BuildKubectlProxyComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Binary:         kubectlPath,
		BindAddress:    conf.BindAddress,
		Port:           conf.KubeApiserverInsecurePort,
		KubeconfigPath: env.inClusterKubeconfigPath,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubectlProxyComponent)
	return nil
}
