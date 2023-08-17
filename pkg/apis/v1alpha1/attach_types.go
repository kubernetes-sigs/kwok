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
	// AttachKind is the kind of the Logs.
	AttachKind = "Attach"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=attaches,verbs=create;delete;get;list;patch;update;watch

// Attach provides attach configuration for a single pod.
type Attach struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for attach
	Spec AttachSpec `json:"spec"`
	// Status holds status for attach
	//+k8s:conversion-gen=false
	Status AttachStatus `json:"status,omitempty"`
}

// AttachStatus holds status for attach
type AttachStatus struct {
	// Conditions holds conditions for attach
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// AttachSpec holds spec for attach.
type AttachSpec struct {
	// Attaches is a list of attaches to configure.
	Attaches []AttachConfig `json:"attaches"`
}

// AttachConfig holds information how to attach.
type AttachConfig struct {
	// Containers is list of container names.
	Containers []string `json:"containers,omitempty"`
	// LogsFile is the file from which the attach starts
	LogsFile *string `json:"logsFile,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// AttachList contains a list of Attach
type AttachList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Attach `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Attach{}, &AttachList{})
}
