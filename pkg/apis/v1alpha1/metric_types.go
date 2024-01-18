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
	// MetricKind is the kind for metrics.
	MetricKind = "Metric"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=metrics,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=metrics/status,verbs=update;patch

// Metric provides metrics configuration.
type Metric struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for metrics.
	Spec MetricSpec `json:"spec"`
	// Status holds status for metrics
	//+k8s:conversion-gen=false
	Status MetricStatus `json:"status,omitempty"`
}

// MetricStatus holds status for metrics
type MetricStatus struct {
	// Conditions holds conditions for metrics.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// MetricSpec holds spec for metrics.
type MetricSpec struct {
	// Path is a restful service path.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path"`
	// Metrics is a list of metric configurations.
	Metrics []MetricConfig `json:"metrics"`
}

// MetricConfig provides metric configuration to a single metric
// +kubebuilder:validation:XValidation:rule="self.kind == 'counter' && self.value.size() != 0",message="counter metric must have value"
// +kubebuilder:validation:XValidation:rule="self.kind == 'gauge' && self.value.size() != 0",message="gauge metric must have value"
// +kubebuilder:validation:XValidation:rule="self.kind == 'histogram' && self.buckets.size() != 0",message="histogram metric must have buckets"
type MetricConfig struct {
	// Name is the fully-qualified name of the metric.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Help provides information about this metric.
	Help string `json:"help,omitempty"`
	// Kind is kind of metric
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=counter;gauge;histogram
	Kind Kind `json:"kind"`
	// Labels are metric labels.
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	Labels []MetricLabel `json:"labels,omitempty"`
	// Value is a CEL expression.
	Value string `json:"value,omitempty"`
	// Buckets is a list of buckets for a histogram metric.
	// +patchMergeKey=le
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=le
	Buckets []MetricBucket `json:"buckets,omitempty"`
	// Dimension is a dimension of the metric.
	// +default="node"
	// +kubebuilder:validation:Enum=node;pod;container
	Dimension Dimension `json:"dimension,omitempty"`
}

// Kind is kind of metric configuration.
// +enum
type Kind string

const (
	// KindCounter is a counter metric.
	KindCounter Kind = "counter"
	// KindGauge is a gauge metric.
	KindGauge Kind = "gauge"
	// KindHistogram is a histogram metric.
	KindHistogram Kind = "histogram"
)

// Dimension is a dimension of the metric.
// +enum
type Dimension string

const (
	// DimensionNode is a node dimension.
	DimensionNode Dimension = "node"
	// DimensionPod is a pod dimension.
	DimensionPod Dimension = "pod"
	// DimensionContainer is a container dimension.
	DimensionContainer Dimension = "container"
)

// MetricLabel holds label name and the value of the label.
type MetricLabel struct {
	// Name is a label name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Value is a CEL expression.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value"`
}

// MetricBucket is a single bucket for a metric.
type MetricBucket struct {
	// Le is less-than or equal.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	Le float64 `json:"le"`
	// Value is a CEL expression.
	// +kubebuilder:validation:Required
	Value string `json:"value"`
	// Hidden is means that this bucket not shown in the metric.
	// but value will be calculated and cumulative into the next bucket.
	Hidden bool `json:"hidden,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// MetricList contains a list of Metric
type MetricList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Metric `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Metric{}, &MetricList{})
}
