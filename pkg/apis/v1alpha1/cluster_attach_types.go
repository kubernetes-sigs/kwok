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
	// ClusterAttachKind is the kind of the ClusterAttachKind.
	ClusterAttachKind = "ClusterAttach"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterattaches,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterattaches/status,verbs=update;patch

// ClusterAttach provides cluster-wide logging configuration
type ClusterAttach struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for cluster attach.
	Spec ClusterAttachSpec `json:"spec"`
	// Status holds status for cluster attach
	//+k8s:conversion-gen=false
	Status ClusterAttachStatus `json:"status,omitempty"`
}

// ClusterAttachStatus holds status for cluster attach
type ClusterAttachStatus struct {
	// Conditions holds conditions for cluster attach.
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterAttachSpec holds spec for cluster attach.
type ClusterAttachSpec struct {
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Attaches is a list of attach configurations.
	Attaches []AttachConfig `json:"attaches"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterAttachList contains a list of ClusterAttach
type ClusterAttachList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterAttach `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterAttach{}, &ClusterAttachList{})
}
