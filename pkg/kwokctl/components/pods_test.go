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
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestConvertToPod(t *testing.T) {
	type args struct {
		component internalversion.Component
	}
	tests := []struct {
		name string
		args args
		want corev1.Pod
	}{
		{
			name: "test1",
			args: args{
				component: internalversion.Component{
					Name:  "n1",
					Image: "i1",

					Envs: []internalversion.Env{
						{
							Name:  "k1",
							Value: "v1",
						},
					},
					Ports: []internalversion.Port{
						{
							Name:     "p1",
							Port:     8080,
							HostPort: 80,
							Protocol: "TCP",
						},
					},
					Volumes: []internalversion.Volume{
						{
							Name:      "v1",
							HostPath:  "/tmp",
							MountPath: "/tmp/host",
							ReadOnly:  false,
						},
					},
				},
			},
			want: corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "n1", Namespace: "kube-system"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "v1", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"}}},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:  format.Ptr[int64](0),
						RunAsGroup: format.Ptr[int64](0),
					},
					Containers: []corev1.Container{
						{
							Name:  "n1",
							Image: "i1",
							Ports: []corev1.ContainerPort{
								{Name: "p1", HostPort: 8080, ContainerPort: 8080, Protocol: "TCP"},
							},

							Env: []corev1.EnvVar{
								{Name: "k1", Value: "v1"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "v1", MountPath: "/tmp/host"},
							},
							ImagePullPolicy: "Never",
						},
					},
					RestartPolicy: "Always",
					HostNetwork:   true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToPod(tt.args.component); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToPod() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertFromPod(t *testing.T) {
	type args struct {
		p corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want internalversion.Component
	}{
		{
			name: "test1",
			args: args{
				p: corev1.Pod{
					TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "n1", Namespace: "kube-system"},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{Name: "v1", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp"}}},
						},
						SecurityContext: &corev1.PodSecurityContext{
							RunAsUser:  format.Ptr[int64](0),
							RunAsGroup: format.Ptr[int64](0),
						},
						Containers: []corev1.Container{
							{
								Name:  "n1",
								Image: "i1",
								Ports: []corev1.ContainerPort{
									{Name: "p1", HostPort: 8080, ContainerPort: 8080, Protocol: "TCP"},
								},

								Env: []corev1.EnvVar{
									{Name: "k1", Value: "v1"},
								},
								VolumeMounts: []corev1.VolumeMount{
									{Name: "v1", MountPath: "/tmp/host"},
								},
								ImagePullPolicy: "Never",
							},
						},
						RestartPolicy: "Always",
						HostNetwork:   true,
					},
				},
			},
			want: internalversion.Component{
				Name:  "n1",
				Image: "i1",

				Envs: []internalversion.Env{
					{
						Name:  "k1",
						Value: "v1",
					},
				},
				Ports: []internalversion.Port{
					{
						Name:     "p1",
						Port:     8080,
						HostPort: 8080,
						Protocol: "TCP",
					},
				},
				Volumes: []internalversion.Volume{
					{
						Name:      "v1",
						HostPath:  "/tmp",
						MountPath: "/tmp/host",
						ReadOnly:  false,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertFromPod(tt.args.p); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertFromPod() = %v, want %v", got, tt.want)
			}
		})
	}
}
