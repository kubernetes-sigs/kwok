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

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addKubeApiserver(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the kube-apiserver
	err = c.EnsureImage(ctx, c.runtime, conf.KubeApiserverImage)
	if err != nil {
		return err
	}
	kubeApiserverVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KubeApiserverImage, consts.ComponentKubeApiserver)
	if err != nil {
		return err
	}

	kubeApiserverTracingConfigPath := ""
	if conf.JaegerPort != 0 {
		kubeApiserverTracingConfigData, err := components.BuildKubeApiserverTracing(components.BuildKubeApiserverTracingConfig{
			Endpoint: c.Name() + "-jaeger:4317",
		})
		if err != nil {
			return fmt.Errorf("failed to generate kubeApiserverTracingConfig yaml: %w", err)
		}
		kubeApiserverTracingConfigPath = c.GetWorkdirPath(runtime.ApiserverTracingConfig)

		err = c.WriteFile(kubeApiserverTracingConfigPath, []byte(kubeApiserverTracingConfigData))
		if err != nil {
			return fmt.Errorf("failed to write kubeApiserverTracingConfig yaml: %w", err)
		}
	}

	kubeApiserverComponent, err := components.BuildKubeApiserverComponent(components.BuildKubeApiserverComponentConfig{
		Runtime:           conf.Runtime,
		ProjectName:       c.Name(),
		Workdir:           env.workdir,
		Image:             conf.KubeApiserverImage,
		Version:           kubeApiserverVersion,
		BindAddress:       utilsnet.PublicAddress,
		Port:              conf.KubeApiserverPort,
		KubeRuntimeConfig: conf.KubeRuntimeConfig,
		KubeFeatureGates:  conf.KubeFeatureGates,
		SecurePort:        conf.SecurePort,
		KubeAuthorization: conf.KubeAuthorization,
		KubeAdmission:     conf.KubeAdmission,
		AuditPolicyPath:   env.auditPolicyPath,
		AuditLogPath:      env.auditLogPath,
		CaCertPath:        env.caCertPath,
		AdminCertPath:     env.adminCertPath,
		AdminKeyPath:      env.adminKeyPath,
		EtcdPort:          conf.EtcdPort,
		EtcdAddress:       c.Name() + "-etcd",
		Verbosity:         env.verbosity,
		DisableQPSLimits:  conf.DisableQPSLimits,
		TracingConfigPath: kubeApiserverTracingConfigPath,
		EtcdPrefix:        conf.EtcdPrefix,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kubeApiserverComponent)
	return nil
}
