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
)

func (c *Cluster) addSchedulerPlugins(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentSchedulerPlugins) {
		return nil
	}

	if !slices.Contains(env.components, consts.ComponentKubeScheduler) {
		return fmt.Errorf("failed to find kube-scheduler component for mutating with scheduler plugins controller")
	}

	conf := &env.kwokctlConfig.Options

	err = c.EnsureImage(ctx, c.runtime, conf.SchedulerPluginsControllerImage)
	if err != nil {
		return err
	}

	version, err := c.ParseVersionFromImage(ctx, c.runtime, conf.SchedulerPluginsControllerImage, "")
	if err != nil {
		return err
	}

	var rawManifests []string
	for _, manifest := range conf.SchedulerPluginsManifests {
		rawManifest, err := c.EnsureManifest(ctx, manifest)
		if err != nil {
			return err
		}
		if len(rawManifest) == 0 {
			continue
		}
		rawManifests = append(rawManifests, string(rawManifest))
	}

	schedulerPluginsControllerComponent, err := components.BuildSchedulerPluginsControllerComponent(components.BuildSchedulerPluginsControllerComponentConfig{
		Runtime:        conf.Runtime,
		Workdir:        env.workdir,
		Image:          conf.SchedulerPluginsControllerImage,
		RawManifests:   rawManifests,
		Version:        version,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
	})
	if err != nil {
		return err
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, schedulerPluginsControllerComponent)
	return nil
}
