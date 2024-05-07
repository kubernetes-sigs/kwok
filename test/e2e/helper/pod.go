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

package helper

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

// PodBuilder is a builder for pod.
type PodBuilder struct {
	pod *corev1.Pod
}

// NewPodBuilder will create a pod builder.
func NewPodBuilder(name string) *PodBuilder {
	return &PodBuilder{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:  "init",
						Image: "image",
					},
					{
						Name:          "sidecar",
						Image:         "image",
						RestartPolicy: format.Ptr(corev1.ContainerRestartPolicyAlways),
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "image",
					},
				},
			},
		},
	}
}

// WithHostNetwork will set host network for pod.
func (b PodBuilder) WithHostNetwork(hostNetwork bool) *PodBuilder {
	b.pod.Spec.HostNetwork = hostNetwork
	return &b
}

// WithNamespace will set namespace for pod.
func (b PodBuilder) WithNamespace(namespace string) *PodBuilder {
	b.pod.ObjectMeta.Namespace = namespace
	return &b
}

// WithNodeName will set node name for pod.
func (b PodBuilder) WithNodeName(nodeName string) *PodBuilder {
	b.pod.Spec.NodeName = nodeName
	return &b
}

// WithAnnotation will set annotation for pod.
func (b PodBuilder) WithAnnotation(key, value string) *PodBuilder {
	if b.pod.ObjectMeta.Annotations == nil {
		b.pod.ObjectMeta.Annotations = map[string]string{}
	}
	b.pod.ObjectMeta.Annotations[key] = value
	return &b
}

// Build will build a pod.
func (b PodBuilder) Build() *corev1.Pod {
	return b.pod.DeepCopy()
}
