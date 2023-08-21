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
	// CustomMetricKind is the kind for resource usage.
	CustomMetricKind = "CustomMetric"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=custommetrics,verbs=create;delete;get;list;patch;update;watch

// CustomMetric provides resource usage for a single pod.
type CustomMetric struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for resource usage.
	Spec CustomMetricSpec `json:"spec"`
	// Status holds status for resource usage
	//+k8s:conversion-gen=false
	Status CustomMetricStatus `json:"status,omitempty"`
}

// CustomMetricStatus holds status for custom metric.
type CustomMetricStatus struct {
	// Conditions holds conditions for custom metric.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// CustomMetricSpec holds spec for custom metric.
type CustomMetricSpec struct {
	// ResourceRef specifies the Kind and version of the resource.
	ResourceRef ResourceRef `json:"resourceRef"`
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Metrics is a list of custom metric.
	Metrics []CustomMetricItem `json:"metrics,omitempty"`
}

// CustomMetricItem holds spec for custom metric item.
type CustomMetricItem struct {
	// Name is the name for custom metric.
	Name string `json:"name"`
	// Value is the value for custom metric.
	Value *resource.Quantity `json:"value,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// CustomMetricList is a list of CustomMetric.
type CustomMetricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CustomMetric `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CustomMetric{}, &CustomMetricList{})
}
