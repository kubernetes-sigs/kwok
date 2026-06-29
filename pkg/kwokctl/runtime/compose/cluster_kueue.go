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

func (c *Cluster) addKueue(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKueue) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kueue
	err = c.EnsureImage(ctx, c.runtime, conf.KueueImage)
	if err != nil {
		return err
	}

	kueueVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KueueImage, "")
	if err != nil {
		return err
	}

	var rawManifests []string
	for _, manifest := range conf.KueueManifests {
		rawManifest, err := c.EnsureManifest(ctx, manifest)
		if err != nil {
			return err
		}
		if len(rawManifest) == 0 {
			continue
		}
		rawManifests = append(rawManifests, string(rawManifest))
	}

	kueueConfigPath := c.GetWorkdirPath(runtime.Kueue)

	if !c.IsDryRun() {
		kueueConfigData, err := components.BuildKueueConfig(rawManifests)
		if err != nil {
			return err
		}
		err = c.WriteFileWithMode(kueueConfigPath, []byte(kueueConfigData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write kueue yaml: %w", err)
		}
	}

	kueueComponent, err := components.BuildKueueComponent(components.BuildKueueComponentConfig{
		Runtime:           conf.Runtime,
		ProjectName:       c.Name(),
		Workdir:           env.workdir,
		Image:             conf.KueueImage,
		RawManifests:      rawManifests,
		Version:           kueueVersion,
		BindAddress:       utilsnet.PublicAddress,
		Port:              conf.MetricsServerPort,
		CaCertPath:        env.caCertPath,
		AdminCertPath:     env.adminCertPath,
		AdminKeyPath:      env.adminKeyPath,
		KubeconfigPath:    env.inClusterOnHostKubeconfigPath,
		ConfigPath:        kueueConfigPath,
		Verbosity:         env.verbosity,
		KubeApiserverPort: conf.KubeApiserverPort,
	})
	if err != nil {
		return err
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kueueComponent)
	return nil
}

func (c *Cluster) addKueueviz(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKueueviz) {
		return nil
	}

	conf := &env.kwokctlConfig.Options
	// Configure the kueueviz backend
	err = c.EnsureImage(ctx, c.runtime, conf.KueuevizBackendImage)
	if err != nil {
		return err
	}

	err = c.EnsureImage(ctx, c.runtime, conf.KueuevizFrontendImage)
	if err != nil {
		return err
	}

	var kueuevizBackendPort uint32
	err = c.setupPorts(ctx,
		env.usedPorts,
		&kueuevizBackendPort,
	)
	if err != nil {
		return err
	}

	kueuevizBackendComponent, err := components.BuildKueuevizBackendComponent(components.BuildKueuevizBackendComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Image:          conf.KueuevizBackendImage,
		Port:           kueuevizBackendPort,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
	})
	if err != nil {
		return err
	}

	kueuevizFrontendComponent, err := components.BuildKueuevizFrontendComponent(components.BuildKueuevizFrontendComponentConfig{
		Runtime:     conf.Runtime,
		ProjectName: c.Name(),
		Image:       conf.KueuevizFrontendImage,
		Port:        conf.KueuevizPort,
		BackendPort: kueuevizBackendPort,
	})
	if err != nil {
		return err
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components,
		kueuevizBackendComponent,
		kueuevizFrontendComponent,
	)
	return nil
}
