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
	"slices"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addDashboard(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentDashboard) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	err = c.EnsureImage(ctx, c.runtime, conf.DashboardImage)
	if err != nil {
		return err
	}
	dashboardVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.DashboardImage, "")
	if err != nil {
		return err
	}

	enableMetricsServer := slices.Contains(env.components, consts.ComponentMetricsServer)
	dashboardComponent, err := components.BuildDashboardComponent(components.BuildDashboardComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.DashboardImage,
		Version:        dashboardVersion,
		BindAddress:    utilsnet.PublicAddress,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		Port:           conf.DashboardPort,
		Banner:         fmt.Sprintf("Welcome to %s", c.Name()),
		EnableMetrics:  enableMetricsServer,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardComponent)

	if enableMetricsServer {
		err = c.EnsureImage(ctx, c.runtime, conf.DashboardMetricsScraperImage)
		if err != nil {
			return err
		}
		dashboardMetricsScraperComponent, err := components.BuildDashboardMetricsScraperComponent(components.BuildDashboardMetricsScraperComponentConfig{
			Runtime:        conf.Runtime,
			Workdir:        env.workdir,
			Image:          conf.DashboardMetricsScraperImage,
			KubeconfigPath: env.inClusterOnHostKubeconfigPath,
			CaCertPath:     env.caCertPath,
			AdminCertPath:  env.adminCertPath,
			AdminKeyPath:   env.adminKeyPath,
		})
		if err != nil {
			return err
		}
		env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, dashboardMetricsScraperComponent)
	}
	return nil
}
