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
	// ClusterExecKind is the kind of the ClusterExec.
	ClusterExecKind = "ClusterExec"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=clusterexecs,verbs=create;delete;get;list;patch;update;watch

// ClusterExec provides cluster-wide exec configuration.
type ClusterExec struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for cluster exec.
	Spec ClusterExecSpec `json:"spec"`
	// Status holds status for cluster exec
	//+k8s:conversion-gen=false
	Status ClusterExecStatus `json:"status,omitempty"`
}

// ClusterExecStatus holds status for cluster exec
type ClusterExecStatus struct {
	// Conditions holds conditions for cluster exec.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterExecSpec holds spec for cluster exec.
type ClusterExecSpec struct {
	// Selector is a selector to filter pods to configure.
	Selector *ObjectSelector `json:"selector,omitempty"`
	// Execs is a list of exec to configure.
	Execs []ExecTarget `json:"execs"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ClusterExecList contains a list of ClusterExec
type ClusterExecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterExec `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterExec{}, &ClusterExecList{})
}
