/*
Copyright 2023 The Kubernetes Authors.

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

package components

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// ConvertToPod converts a component to a pod.
func ConvertToPod(component internalversion.Component) corev1.Pod {
	env := []corev1.EnvVar{}
	for _, e := range component.Envs {
		env = append(env, corev1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	ports := []corev1.ContainerPort{}
	for _, p := range component.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: int32(p.Port),
			HostPort:      int32(p.Port), // same as container port because of host network
			Protocol:      corev1.Protocol(p.Protocol),
		})
	}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	for i, v := range component.Volumes {
		name := v.Name
		if name == "" {
			name = fmt.Sprintf("volume-%d", i)
		}
		s := corev1.HostPathVolumeSource{
			Path: v.HostPath,
		}
		if v.PathType != "" {
			t := corev1.HostPathType(v.PathType)
			s.Type = &t
		}
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &s,
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      name,
			MountPath: v.MountPath,
			ReadOnly:  v.ReadOnly,
		})
	}

	p := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      component.Name,
			Namespace: metav1.NamespaceSystem,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            component.Name,
					Image:           component.Image,
					ImagePullPolicy: corev1.PullNever,
					Args:            component.Args,
					Command:         component.Command,
					Env:             env,
					Ports:           ports,
					VolumeMounts:    volumeMounts,
				},
			},
			Volumes: volumes,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsGroup: format.Ptr[int64](0),
				RunAsUser:  format.Ptr[int64](0),
			},
			RestartPolicy: corev1.RestartPolicyAlways,
			HostNetwork:   true,
		},
	}
	return p
}

// ConvertFromPod converts a pod to a component.
func ConvertFromPod(p corev1.Pod) internalversion.Component {
	if len(p.Spec.Containers) == 0 {
		return internalversion.Component{}
	}
	ports := []internalversion.Port{}
	container := p.Spec.Containers[0]

	for _, port := range container.Ports {
		ports = append(ports, internalversion.Port{
			Name:     port.Name,
			Port:     uint32(port.ContainerPort),
			HostPort: uint32(port.HostPort),
			Protocol: internalversion.Protocol(port.Protocol),
		})
	}

	volumes := []internalversion.Volume{}

	for _, vm := range container.VolumeMounts {
		v, ok := slices.Find(p.Spec.Volumes, func(v corev1.Volume) bool {
			return v.Name == vm.Name
		})
		if !ok {
			continue
		}
		if v.HostPath == nil {
			continue
		}
		volumes = append(volumes, internalversion.Volume{
			Name:      v.Name,
			HostPath:  v.HostPath.Path,
			PathType:  internalversion.HostPathType(format.ElemOrDefault(v.HostPath.Type)),
			MountPath: vm.MountPath,
			ReadOnly:  vm.ReadOnly,
		})
	}

	envs := []internalversion.Env{}
	for _, e := range p.Spec.Containers[0].Env {
		envs = append(envs, internalversion.Env{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	return internalversion.Component{
		Name:    p.Name,
		Image:   container.Image,
		Args:    container.Args,
		Command: container.Command,
		Ports:   ports,
		Volumes: volumes,
		Envs:    envs,
	}
}
