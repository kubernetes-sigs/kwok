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

// Metric provides metrics configuration.
type Metric struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for metrics.
	Spec MetricSpec `json:"spec"`
}

// MetricSpec holds spec for metrics.
type MetricSpec struct {
	// Path is a restful service path.
	Path string `json:"path"`
	// Metrics is a list of metric configurations.
	Metrics []MetricConfig `json:"metrics"`
}

// MetricConfig provides metric configuration to a single metric
type MetricConfig struct {
	// Name is the fully-qualified name of the metric.
	Name string `json:"name"`
	// Help provides information about this metric.
	Help string `json:"help"`
	// Kind is kind of metric (ex. counter, gauge, histogram).
	Kind string `json:"kind"`
	// Labels are metric labels.
	Labels []MetricLabel `json:"labels"`
	// Value is a CEL expression.
	Value string `json:"value"`
	// Buckets is a list of buckets for a histogram metric.
	Buckets []MetricBucket `json:"buckets"`
}

// MetricLabel holds label name and the value of the label.
type MetricLabel struct {
	// Name is a label name.
	Name string `json:"name"`
	// Value is a CEL expression.
	Value string `json:"value"`
}

// MetricBucket is a single bucket for a metric.
type MetricBucket struct {
	// Le is less-than or equal.
	Le float64 `json:"le"`
	// Value is a CEL expression.
	Value string `json:"value"`
	// Hidden is means that this bucket not shown in the metric.
	// but value will be calculated and cumulative into the next bucket.
	Hidden bool `json:"hidden"`
}
