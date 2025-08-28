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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourceUsageKind is the kind for resource usage.
	ResourceUsageKind = "ResourceUsage"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=resourceusages,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=resourceusages/status,verbs=update;patch

// ResourceUsage provides resource usage for a single pod.
type ResourceUsage struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for resource usage.
	Spec ResourceUsageSpec `json:"spec"`
	// Status holds status for resource usage
	//+k8s:conversion-gen=false
	Status ResourceUsageStatus `json:"status,omitempty"`
}

// ResourceUsageStatus holds status for resource usage
type ResourceUsageStatus struct {
	// Conditions holds conditions for resource usage
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ResourceUsageSpec holds spec for resource usage.
type ResourceUsageSpec struct {
	// Usages is a list of resource usage for the pod.
	Usages []ResourceUsageContainer `json:"usages,omitempty"`
}

// ResourceUsageContainer holds spec for resource usage container.
type ResourceUsageContainer struct {
	// Containers is list of container names.
	Containers []string `json:"containers,omitempty"`
	// Usage is a list of resource usage for the container.
	Usage map[string]ResourceUsageValue `json:"usage,omitempty"`
}

// ResourceUsageValue holds value for resource usage.
type ResourceUsageValue struct {
	// Value is the value for resource usage.
	Value *resource.Quantity `json:"value,omitempty"`
	// Expression is the expression for resource usage.
	Expression *string `json:"expression,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ResourceUsageList is a list of ResourceUsage.
type ResourceUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ResourceUsage `json:"items"`
}
