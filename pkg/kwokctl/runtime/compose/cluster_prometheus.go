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
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) setupPrometheusConfig(_ context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentPrometheus) {
		return nil
	}

	// Configure the prometheus
	prometheusData, err := components.BuildPrometheus(components.BuildPrometheusConfig{
		Components: env.kwokctlConfig.Components,
	})
	if err != nil {
		return fmt.Errorf("failed to generate prometheus yaml: %w", err)
	}
	prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

	// We don't need to check the permissions of the prometheus config file,
	// because it's working in a non-root container.
	err = c.WriteFileWithMode(prometheusConfigPath, []byte(prometheusData), 0644)
	if err != nil {
		return fmt.Errorf("failed to write prometheus yaml: %w", err)
	}
	return nil
}

func (c *Cluster) addPrometheus(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentPrometheus) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the prometheus
	err = c.EnsureImage(ctx, c.runtime, conf.PrometheusImage)
	if err != nil {
		return err
	}

	prometheusVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.PrometheusImage, "")
	if err != nil {
		return err
	}

	prometheusConfigPath := c.GetWorkdirPath(runtime.Prometheus)

	prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
		Runtime:                      conf.Runtime,
		Workdir:                      env.workdir,
		Image:                        conf.PrometheusImage,
		Version:                      prometheusVersion,
		BindAddress:                  utilsnet.PublicAddress,
		Port:                         conf.PrometheusPort,
		ConfigPath:                   prometheusConfigPath,
		AdminCertPath:                env.adminCertPath,
		AdminKeyPath:                 env.adminKeyPath,
		Verbosity:                    env.verbosity,
		DisableKubeControllerManager: !slices.Contains(env.components, consts.ComponentKubeControllerManager),
		DisableKubeScheduler:         !slices.Contains(env.components, consts.ComponentKubeScheduler),
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, prometheusComponent)
	return nil
}
