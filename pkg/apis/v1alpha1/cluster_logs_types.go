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
	// ClusterLogsKind is the kind of the ClusterLogsKind.
	ClusterLogsKind = "ClusterLogs"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterLogs provides cluster-wide logging configuration
type ClusterLogs struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for cluster logs.
	Spec ClusterLogsSpec `json:"spec"`
}

// ClusterLogsSpec holds spec for cluster logs.
type ClusterLogsSpec struct {
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Forwards is a list of log configurations.
	Logs []Log `json:"logs"`
}
