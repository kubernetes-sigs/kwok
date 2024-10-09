/*
Copyright 2024 The Kubernetes Authors.

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
	// ControllerKind is the kind of the Logs.
	ControllerKind = "Controller"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=controllers,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=controllers/status,verbs=update;patch

// Controller provides controller configuration for a single pod.
type Controller struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for controller
	Spec ControllerSpec `json:"spec"`
	// Status holds status for controller
	//+k8s:conversion-gen=false
	Status ControllerStatus `json:"status,omitempty"`
}

// ControllerStatus holds status for controller
type ControllerStatus struct {
	// Conditions holds conditions for controller
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ControllerSpec holds spec for controller.
type ControllerSpec struct {
	Manages []ManagesSelector `json:"manages,omitempty"`
}

// ManagesSelector holds information about the manages selector.
type ManagesSelector struct {
	// Kind of the referent.
	Kind string `json:"kind"`
	// Group of the referent.
	Group string `json:"group,omitempty"`
	// Version of the referent.
	Version string `json:"version,omitempty"`
	// Namespace of the referent. only valid if it is a namespaced.
	Namespace string `json:"namespace,omitempty"`
	// Name of the referent. specify only this one.
	Name string `json:"name,omitempty"`
	// Labels of the referent. specify matched with labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations of the referent. specify matched with annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ControllerList contains a list of Controller
type ControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Controller `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Controller{}, &ControllerList{})
}
