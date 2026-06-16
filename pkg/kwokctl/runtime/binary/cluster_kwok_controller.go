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

package binary

import (
	"context"
	"slices"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKwokController) {
		return nil
	}

	conf := &env.kwokctlConfig.Options

	// Configure the kwok-controller
	kwokControllerPath, err := c.EnsureBinary(ctx, consts.ComponentKwokController, conf.KwokControllerBinary)
	if err != nil {
		return err
	}

	kwokControllerVersion, err := c.ParseVersionFromBinary(ctx, kwokControllerPath)
	if err != nil {
		return err
	}

	otlpGrpcAddress := ""
	if conf.JaegerOtlpGrpcPort != 0 {
		otlpGrpcAddress = utilsnet.LocalAddress + ":" + format.String(conf.JaegerOtlpGrpcPort)
	}

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                  conf.Runtime,
		ProjectName:              c.Name(),
		Workdir:                  env.workdir,
		Binary:                   kwokControllerPath,
		Version:                  kwokControllerVersion,
		BindAddress:              conf.BindAddress,
		Port:                     conf.KwokControllerPort,
		ConfigPath:               env.kwokConfigPath,
		KubeconfigPath:           env.inClusterKubeconfigPath,
		CaCertPath:               env.caCertPath,
		AdminCertPath:            env.adminCertPath,
		AdminKeyPath:             env.adminKeyPath,
		NodeName:                 "localhost",
		Verbosity:                env.verbosity,
		NodeLeaseDurationSeconds: conf.NodeLeaseDurationSeconds,
		EnableCRDs:               conf.EnableCRDs,
		OtlpGrpcAddress:          otlpGrpcAddress,
	})
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}
