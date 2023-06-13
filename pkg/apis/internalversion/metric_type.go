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

package internalversion

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Metric provides metrics configuration.
type Metric struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for metrics.
	Spec MetricSpec
}

// MetricSpec holds spec for metrics.
type MetricSpec struct {
	// Path is a restful service path.
	Path string
	// Metrics is a list of metric configurations.
	Metrics []MetricConfig
}

// MetricConfig provides metric configuration to a single metric
type MetricConfig struct {
	// Name is the fully-qualified name of the metric.
	Name string
	// Help provides information about this metric.
	Help string
	// Kind is kind of metric (ex. counter, gauge, histogram).
	Kind string
	// Labels are metric labels.
	Labels []MetricLabel
	// Value is a CEL expression.
	Value string
	// Buckets is a list of buckets for a histogram metric.
	Buckets []MetricBucket
}

// MetricLabel holds label name and the value of the label.
type MetricLabel struct {
	// Name is a label name.
	Name string
	// Value is a CEL expression.
	Value string
}

// MetricBucket is a single bucket for a metric.
type MetricBucket struct {
	// Le is less-than or equal.
	Le float64
	// Value is a CEL expression.
	Value string
	// Hidden is means that this bucket not shown in the metric.
	// but value will be calculated and cumulative into the next bucket.
	Hidden bool
}
