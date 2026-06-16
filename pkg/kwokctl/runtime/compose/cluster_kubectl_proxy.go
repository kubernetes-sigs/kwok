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
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addKubectlProxy(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKubeApiserverInsecureProxy) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kubectl
	err = c.EnsureImage(ctx, c.runtime, conf.KubectlImage)
	if err != nil {
		return err
	}

	kubectlProxyComponent, err := components.BuildKubectlProxyComponent(components.BuildKubectlProxyComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.KubectlImage,
		BindAddress:    utilsnet.PublicAddress,
		Port:           conf.KubeApiserverInsecurePort,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
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
