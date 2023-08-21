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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ExternalMetricKind is the kind for resource usage.
	ExternalMetricKind = "ExternalMetric"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=externalmetrics,verbs=create;delete;get;list;patch;update;watch

// ExternalMetric provides resource usage for a single pod.
type ExternalMetric struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for resource usage.
	Spec ExternalMetricSpec `json:"spec"`
	// Status holds status for resource usage
	//+k8s:conversion-gen=false
	Status ExternalMetricStatus `json:"status,omitempty"`
}

// ExternalMetricStatus holds status for external metric.
type ExternalMetricStatus struct {
	// Conditions holds conditions for external metric.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ExternalMetricSpec holds spec for external metric.
type ExternalMetricSpec struct {
	// Name is the name of external metric.
	Name string `json:"name,omitempty"`
	// Metrics is a list of external metric.
	Metrics []ExternalMetricItem `json:"metrics,omitempty"`
}

// ExternalMetricItem holds spec for external metric item.
type ExternalMetricItem struct {
	// Value is the value for external metric.
	Value *resource.Quantity `json:"value,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ExternalMetricList is a list of ExternalMetric.
type ExternalMetricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ExternalMetric `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalMetric{}, &ExternalMetricList{})
}
