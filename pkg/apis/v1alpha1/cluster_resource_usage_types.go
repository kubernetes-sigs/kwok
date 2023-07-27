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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterResourceUsageKind is the kind for ClusterResourceUsage.
	ClusterResourceUsageKind = "ClusterResourceUsage"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterresourceusages,verbs=create;delete;get;list;patch;update;watch

// ClusterResourceUsage provides cluster-wide resource usage.
type ClusterResourceUsage struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for cluster resource usage.
	Spec ClusterResourceUsageSpec `json:"spec"`
	// Status holds status for cluster resource usage
	//+k8s:conversion-gen=false
	Status ClusterResourceUsageStatus `json:"status,omitempty"`
}

// ClusterResourceUsageStatus holds status for cluster resource usage
type ClusterResourceUsageStatus struct {
	// Conditions holds conditions for cluster resource usage
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterResourceUsageSpec holds spec for cluster resource usage.
type ClusterResourceUsageSpec struct {
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Usages is a list of resource usage for the pod.
	Usages []ResourceUsageContainer `json:"usages,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterResourceUsageList is a list of ClusterResourceUsage.
type ClusterResourceUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ClusterResourceUsage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterResourceUsage{}, &ClusterResourceUsageList{})
}
