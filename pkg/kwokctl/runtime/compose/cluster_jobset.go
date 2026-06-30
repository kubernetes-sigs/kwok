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

func (c *Cluster) addJobSet(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentJobSet) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the jobset
	err = c.EnsureImage(ctx, c.runtime, conf.JobSetImage)
	if err != nil {
		return err
	}

	jobsetVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.JobSetImage, "")
	if err != nil {
		return err
	}

	var rawManifests []string
	for _, manifest := range conf.JobSetManifests {
		rawManifest, err := c.EnsureManifest(ctx, manifest)
		if err != nil {
			return err
		}
		if len(rawManifest) == 0 {
			continue
		}
		rawManifests = append(rawManifests, string(rawManifest))
	}

	jobsetConfigPath := c.GetWorkdirPath(runtime.JobSet)

	if !c.IsDryRun() {
		jobsetConfigData, err := components.BuildJobSetConfig(rawManifests)
		if err != nil {
			return err
		}

		err = c.WriteFile(jobsetConfigPath, []byte(jobsetConfigData))
		if err != nil {
			return fmt.Errorf("failed to write jobset yaml: %w", err)
		}
	}

	jobsetComponent, err := components.BuildJobSetComponent(components.BuildJobSetComponentConfig{
		Runtime:        conf.Runtime,
		ProjectName:    c.Name(),
		Workdir:        env.workdir,
		Image:          conf.JobSetImage,
		RawManifests:   rawManifests,
		Version:        jobsetVersion,
		BindAddress:    utilsnet.PublicAddress,
		CaCertPath:     env.caCertPath,
		AdminCertPath:  env.adminCertPath,
		AdminKeyPath:   env.adminKeyPath,
		ConfigPath:     jobsetConfigPath,
		KubeconfigPath: env.inClusterOnHostKubeconfigPath,
		Verbosity:      env.verbosity,
	})
	if err != nil {
		return err
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, jobsetComponent)
	return nil
}
