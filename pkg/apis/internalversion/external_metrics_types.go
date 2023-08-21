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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExternalMetric provides resource usage for a single pod.
type ExternalMetric struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for resource usage.
	Spec ExternalMetricSpec
}

// ExternalMetricSpec holds spec for external metric.
type ExternalMetricSpec struct {
	// Name is the name of external metric.
	Name string
	// Metrics is a list of external metric.
	Metrics []ExternalMetricItem
}

// ExternalMetricItem holds spec for external metric item.
type ExternalMetricItem struct {
	// Value is the value for external metric.
	Value *resource.Quantity
}
