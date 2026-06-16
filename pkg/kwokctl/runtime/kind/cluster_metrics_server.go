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

package kind

import (
	"context"
	"fmt"
	"slices"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func (c *Cluster) addMetricsServer(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentMetricsServer) {
		return nil
	}

	conf := &env.kwokctlConfig.Options
	err = c.EnsureImage(ctx, c.runtime, conf.MetricsServerImage)
	if err != nil {
		return err
	}
	metricsServerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.MetricsServerImage, "")
	if err != nil {
		return err
	}

	var rawManifests []string
	for _, manifest := range conf.MetricsServerManifests {
		rawManifest, err := c.EnsureManifest(ctx, manifest)
		if err != nil {
			return err
		}
		if len(rawManifest) == 0 {
			continue
		}
		rawManifests = append(rawManifests, string(rawManifest))
	}

	metricsServerComponent, err := components.BuildMetricsServerComponent(components.BuildMetricsServerComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.MetricsServerImage,
		RawManifests:   rawManifests,
		Version:        metricsServerVersion,
		BindAddress:    utilsnet.PublicAddress,
		Port:           443,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}

	metricsServerPod, err := yaml.Marshal(components.ConvertToPod(metricsServerComponent))
	if err != nil {
		return fmt.Errorf("failed to marshal metrics server pod: %w", err)
	}
	err = c.WriteFile(utilspath.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentMetricsServer+".yaml"), metricsServerPod)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, metricsServerComponent)

	return nil
}
