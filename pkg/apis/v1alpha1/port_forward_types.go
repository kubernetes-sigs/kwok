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
	// PortForwardKind is the kind of the PortForward.
	PortForwardKind = "PortForward"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=portforwards,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=portforwards/status,verbs=update;patch

// PortForward provides port forward configuration for a single pod.
type PortForward struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for port forward.
	Spec PortForwardSpec `json:"spec"`
	// Status holds status for port forward
	//+k8s:conversion-gen=false
	Status PortForwardStatus `json:"status,omitempty"`
}

// PortForwardStatus holds status for port forward
type PortForwardStatus struct {
	// Conditions holds conditions for port forward
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// PortForwardSpec holds spec for port forward.
type PortForwardSpec struct {
	// Forwards is a list of forwards to configure.
	Forwards []Forward `json:"forwards"`
}

// Forward holds information how to forward based on ports.
// +kubebuilder:validation:XValidation:rule="has(self.target) != has(self.command)",message="target and command can't be set at the same time"
type Forward struct {
	// Ports is a list of ports to forward.
	// if not set, all ports will be forwarded.
	Ports []int32 `json:"ports,omitempty"`
	// Target is the target to forward to.
	Target *ForwardTarget `json:"target,omitempty"`
	// Command is the command to run to forward with stdin/stdout.
	// if set, Target will be ignored.
	Command []string `json:"command,omitempty"`
}

// ForwardTarget holds information how to forward to a target.
type ForwardTarget struct {
	// Port is the port to forward to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
	// Address is the address to forward to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Address string `json:"address"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// PortForwardList contains a list of PortForward
type PortForwardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PortForward `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PortForward{}, &PortForwardList{})
}
