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

func (c *Cluster) addDescheduler(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentDescheduler) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the descheduler
	err = c.EnsureImage(ctx, c.runtime, conf.DeschedulerImage)
	if err != nil {
		return err
	}

	deschedulerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.DeschedulerImage, "")
	if err != nil {
		return err
	}

	var rawManifests []string
	for _, manifest := range conf.DeschedulerManifests {
		rawManifest, err := c.EnsureManifest(ctx, manifest)
		if err != nil {
			return err
		}
		if len(rawManifest) == 0 {
			continue
		}
		rawManifests = append(rawManifests, string(rawManifest))
	}

	deschedulerConfigPath := c.GetWorkdirPath(runtime.Descheduler)

	if !c.IsDryRun() {
		deschedulerPolicyData, err := components.BuildDeschedulerPolicy(rawManifests)
		if err != nil {
			return err
		}

		err = c.WriteFileWithMode(deschedulerConfigPath, []byte(deschedulerPolicyData), 0644)
		if err != nil {
			return fmt.Errorf("failed to write descheduler yaml: %w", err)
		}
	}

	deschedulerComponent, err := components.BuildDeschedulerComponent(components.BuildDeschedulerComponentConfig{
		Runtime:        conf.Runtime,
		Workdir:        env.workdir,
		Image:          conf.DeschedulerImage,
		RawManifests:   rawManifests,
		Version:        deschedulerVersion,
		BindAddress:    utilsnet.PublicAddress,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		ConfigPath:     deschedulerConfigPath,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, deschedulerComponent)
	return nil
}
