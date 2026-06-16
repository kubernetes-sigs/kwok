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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func (c *Cluster) addKwokController(ctx context.Context, env *env) (err error) {
	if !slices.Contains(env.components, consts.ComponentKwokController) {
		return nil
	}

	conf := &env.kwokctlConfig.Options
	err = c.EnsureImage(ctx, c.runtime, conf.KwokControllerImage)
	if err != nil {
		return err
	}

	// Configure the kwok-controller
	kwokControllerVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.KwokControllerImage, "kwok")
	if err != nil {
		return err
	}

	logVolumes := runtime.GetLogVolumes(ctx)
	logVolumes = utilsslices.Map(logVolumes, func(v internalversion.Volume) internalversion.Volume {
		v.HostPath = utilspath.Join("/var/components/controller", v.HostPath)
		return v
	})

	otlpGrpcAddress := ""
	if conf.JaegerPort != 0 {
		otlpGrpcAddress = utilsnet.LocalAddress + ":4317"
	}

	kwokControllerComponent := components.BuildKwokControllerComponent(components.BuildKwokControllerComponentConfig{
		Runtime:                           conf.Runtime,
		ProjectName:                       c.Name(),
		Workdir:                           env.workdir,
		Image:                             conf.KwokControllerImage,
		Version:                           kwokControllerVersion,
		BindAddress:                       utilsnet.PublicAddress,
		Port:                              conf.KwokControllerPort,
		ConfigPath:                        env.kwokConfigPath,
		KubeconfigPath:                    env.inClusterOnHostKubeconfigPath,
		CaCertPath:                        env.caCertPath,
		AdminCertPath:                     env.adminCertPath,
		AdminKeyPath:                      env.adminKeyPath,
		NodeIP:                            "$(POD_IP)",
		NodeName:                          "kwok-controller.kube-system.svc",
		ManageNodesWithAnnotationSelector: "kwok.x-k8s.io/node=fake",
		Verbosity:                         env.verbosity,
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        conf.EnableCRDs,
		OtlpGrpcAddress:                   otlpGrpcAddress,
	})
	kwokControllerComponent.Volumes = append(kwokControllerComponent.Volumes, logVolumes...)

	runtime.ApplyComponentPatches(ctx, &kwokControllerComponent, env.kwokctlConfig.ComponentsPatches)

	pod := components.ConvertToPod(kwokControllerComponent)
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
	kwokControllerPod, err := yaml.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal kwok controller pod: %w", err)
	}
	err = c.WriteFile(utilspath.Join(c.GetWorkdirPath(runtime.ManifestsName), consts.ComponentKwokController+".yaml"), kwokControllerPod)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, kwokControllerComponent)
	return nil
}
