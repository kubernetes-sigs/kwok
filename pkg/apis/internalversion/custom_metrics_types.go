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

// CustomMetric provides resource usage for a single pod.
type CustomMetric struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for resource usage.
	Spec CustomMetricSpec
}

// CustomMetricSpec holds spec for custom metric.
type CustomMetricSpec struct {
	// ResourceRef specifies the Kind and version of the resource.
	ResourceRef ResourceRef
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector
	// Metrics is a list of custom metric.
	Metrics []CustomMetricItem
}

// CustomMetricItem holds spec for custom metric item.
type CustomMetricItem struct {
	// Name is the name for custom metric.
	Name string
	// Value is the value for custom metric.
	Value *resource.Quantity
}
