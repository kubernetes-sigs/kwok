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
	// LogsKind is the kind of the Logs.
	LogsKind = "Logs"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=logs,verbs=create;delete;get;list;patch;update;watch

// Logs provides logging configuration for a single pod.
type Logs struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for logs
	Spec LogsSpec `json:"spec"`
	// Status holds status for logs
	//+k8s:conversion-gen=false
	Status LogsStatus `json:"status,omitempty"`
}

// LogsStatus holds status for logs
type LogsStatus struct {
	// Conditions holds conditions for logs
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// LogsSpec holds spec for logs.
type LogsSpec struct {
	// Logs is a list of logs to configure.
	Logs []Log `json:"logs"`
}

// Log holds information how to forward logs.
type Log struct {
	// Containers is list of container names.
	Containers []string `json:"containers"`
	// LogsFile is the file from which the log forward starts
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	LogsFile string `json:"logsFile"`
	// Follow up if true
	Follow bool `json:"follow"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// LogsList contains a list of Logs
type LogsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Logs `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Logs{}, &LogsList{})
}
