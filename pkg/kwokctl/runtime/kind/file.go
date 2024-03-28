/*
Copyright 2024 The Kubernetes Authors.

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

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func (c *Cluster) setupFiles(_ context.Context, env *env) (err error) {
	r := gotpl.NewRenderer(nil)
	pathDir := c.GetWorkdirPath("files")
	for index, component := range env.kwokctlConfig.Components {
		for i, file := range component.Files {
			var data []byte
			if file.Data != "" {
				data = []byte(file.Data)
			} else if file.Template != "" {
				d, err := r.ToText(file.Template, env.kwokctlConfig)
				if err != nil {
					return err
				}
				data = d
			}

			filePath := path.Join(pathDir, component.Name, file.Path)

			err = c.MkdirAll(path.Dir(filePath))
			if err != nil {
				return err
			}
			wc, err := c.OpenFile(filePath)
			if err != nil {
				return err
			}
			_, err = wc.Write(data)
			if err != nil {
				return err
			}

			err = wc.Close()
			if err != nil {
				return err
			}

			v := internalversion.Volume{
				Name:      "file-" + format.String(i),
				MountPath: file.Path,
				HostPath:  filePath,
			}

			env.kwokctlConfig.Components[index].Volumes = append(env.kwokctlConfig.Components[index].Volumes, v)
		}
	}

	return nil
}

func (c *Cluster) setupManifests(_ context.Context, env *env) (err error) {
	manifestsDir := c.GetWorkdirPath(runtime.ManifestsName)
	for _, component := range env.kwokctlConfig.Components {
		if component.Name == consts.ComponentEtcd ||
			component.Name == consts.ComponentKubeApiserver ||
			component.Name == consts.ComponentKubeControllerManager ||
			component.Name == consts.ComponentKubeScheduler {
			continue
		}

		component := component
		component.Volumes = slices.Map(component.Volumes, func(v internalversion.Volume) internalversion.Volume {
			v.HostPath = path.Join("/var/components", component.Name, v.MountPath)
			return v
		})

		component.Volumes = append(component.Volumes,
			internalversion.Volume{
				HostPath:  "/etc/kubernetes/pki",
				MountPath: "/etc/kubernetes/pki",
			},
			internalversion.Volume{
				HostPath:  "/etc/kubernetes/admin.conf",
				MountPath: "/root/.kube/config",
			},
		)

		if component.Name == consts.ComponentKwokController {
			component.Volumes = append(component.Volumes,
				internalversion.Volume{
					HostPath:  "/etc/kwok/kwok.yaml",
					MountPath: "/root/.kwok/kwok.yaml",
				},
			)
		}

		pod := components.ConvertToPod(component)
		if component.Name == consts.ComponentKwokController {
			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			})
		}

		componentPod, err := yaml.Marshal(components.ConvertToPod(component))
		if err != nil {
			return fmt.Errorf("failed to marshal pod: %w", err)
		}

		if err != nil {
			return err
		}
		podFile := component.Name + ".yaml"
		err = c.WriteFile(path.Join(manifestsDir, podFile), componentPod)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", podFile, err)
		}
	}
	return nil
}
