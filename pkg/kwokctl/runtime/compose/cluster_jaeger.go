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

func (c *Cluster) addJaeger(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentJaeger) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the jaeger
	err = c.EnsureImage(ctx, c.runtime, conf.JaegerImage)
	if err != nil {
		return err
	}

	jaegerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.JaegerImage, "")
	if err != nil {
		return err
	}

	jaegerComponent, err := components.BuildJaegerComponent(components.BuildJaegerComponentConfig{
		Runtime:      conf.Runtime,
		Workdir:      env.workdir,
		Image:        conf.JaegerImage,
		Version:      jaegerVersion,
		BindAddress:  utilsnet.PublicAddress,
		Port:         conf.JaegerPort,
		OtlpGrpcPort: conf.JaegerOtlpGrpcPort,
		Verbosity:    env.verbosity,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, jaegerComponent)
	return nil
}
