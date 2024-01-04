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
	// ClusterPortForwardKind is the kind of the ClusterPortForward.
	ClusterPortForwardKind = "ClusterPortForward"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterportforwards,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterportforwards/status,verbs=update;patch

// ClusterPortForward provides cluster-wide port forward configuration.
type ClusterPortForward struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for cluster port forward.
	Spec ClusterPortForwardSpec `json:"spec"`
	// Status holds status for cluster port forward
	//+k8s:conversion-gen=false
	Status ClusterPortForwardStatus `json:"status,omitempty"`
}

// ClusterPortForwardStatus holds status for cluster port forward
type ClusterPortForwardStatus struct {
	// Conditions holds conditions for cluster port forward.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterPortForwardSpec holds spec for cluster port forward.
type ClusterPortForwardSpec struct {
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Forwards is a list of forwards to configure.
	Forwards []Forward `json:"forwards"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterPortForwardList contains a list of ClusterPortForward
type ClusterPortForwardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPortForward `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterPortForward{}, &ClusterPortForwardList{})
}
