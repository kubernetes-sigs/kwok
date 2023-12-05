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
)

// NodeBuilder is a builder to build a node.
type NodeBuilder struct {
	node *corev1.Node
}

// NewNodeBuilder will create a node builder.
func NewNodeBuilder(name string) *NodeBuilder {
	return &NodeBuilder{
		node: &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Annotations: map[string]string{
					"kwok.x-k8s.io/node": "fake",
				},
				Labels: map[string]string{
					"type": "kwok",
				},
			},
			Spec: corev1.NodeSpec{
				PodCIDR: "10.10.0.1/24",
			},
		},
	}
}

// WithPodCIDR will set podCIDR for node.
func (b NodeBuilder) WithPodCIDR(podCIDR string) *NodeBuilder {
	b.node.Spec.PodCIDR = podCIDR
	return &b
}

// Build will build a node.
func (b NodeBuilder) Build() *corev1.Node {
	return b.node.DeepCopy()
}
