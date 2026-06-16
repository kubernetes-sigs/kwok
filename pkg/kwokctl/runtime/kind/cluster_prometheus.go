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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
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
	err = c.WriteFile(prometheusConfigPath, []byte(prometheusData))
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

	err = c.EnsureImage(ctx, c.runtime, conf.PrometheusImage)
	if err != nil {
		return err
	}
	prometheusVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.PrometheusImage, "")
	if err != nil {
		return err
	}

	prometheusComponent, err := components.BuildPrometheusComponent(components.BuildPrometheusComponentConfig{
		Runtime:                      conf.Runtime,
		Workdir:                      env.workdir,
		Image:                        conf.PrometheusImage,
		Version:                      prometheusVersion,
		BindAddress:                  utilsnet.PublicAddress,
		Port:                         9090,
		ConfigPath:                   "/var/components/prometheus/etc/prometheus/prometheus.yaml",
		AdminCertPath:                env.adminCertPath,
		AdminKeyPath:                 env.adminKeyPath,
		Verbosity:                    env.verbosity,
		DisableKubeControllerManager: !slices.Contains(env.components, consts.ComponentKubeControllerManager),
		DisableKubeScheduler:         !slices.Contains(env.components, consts.ComponentKubeScheduler),
	})
	if err != nil {
		return err
	}

	prometheusComponent.Volumes = append(prometheusComponent.Volumes,
		internalversion.Volume{
			HostPath:  "/etc/kubernetes/pki/apiserver-etcd-client.crt",
			MountPath: "/etc/kubernetes/pki/apiserver-etcd-client.crt",
			ReadOnly:  true,
		},
		internalversion.Volume{
			HostPath:  "/etc/kubernetes/pki/apiserver-etcd-client.key",
			MountPath: "/etc/kubernetes/pki/apiserver-etcd-client.key",
			ReadOnly:  true,
		},
	)

	runtime.ApplyComponentPatches(ctx, &prometheusComponent, env.kwokctlConfig.ComponentsPatches)

	prometheusPod, err := yaml.Marshal(components.ConvertToPod(prometheusComponent))
	if err != nil {
		return fmt.Errorf("failed to marshal prometheus pod: %w", err)
	}
	err = c.WriteFile(utilspath.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentPrometheus+".yaml"), prometheusPod)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, prometheusComponent)
	return nil
}
