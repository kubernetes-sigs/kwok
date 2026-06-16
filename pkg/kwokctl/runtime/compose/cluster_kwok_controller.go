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
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKwokController) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kwok-controller
	err = c.EnsureImage(ctx, c.runtime, conf.KwokControllerImage)
	if err != nil {
		return err
	}

	kwokControllerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	otlpGrpcAddress := ""
	if conf.JaegerPort != 0 {
		otlpGrpcAddress = c.Name() + "-jaeger:4317"
	}

	logVolumes := runtime.GetLogVolumes(ctx)

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                  conf.Runtime,
		ProjectName:              c.Name(),
		Workdir:                  env.workdir,
		Image:                    conf.KwokControllerImage,
		Version:                  kwokControllerVersion,
		BindAddress:              utilsnet.PublicAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               env.kwokConfigPath,
		KubeconfigPath:           env.inClusterOnHostKubeconfigPath,
		CaCertPath:               env.caCertPath,
		AdminCertPath:            env.adminCertPath,
		AdminKeyPath:             env.adminKeyPath,
		NodeName:                 c.Name() + "-kwok-controller",
		Verbosity:                env.verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		EnableCRDs:               conf.EnableCRDs,
		OtlpGrpcAddress:          otlpGrpcAddress,
	})
	kwokControllerComponent.Volumes = append(kwokControllerComponent.Volumes, logVolumes...)

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}
