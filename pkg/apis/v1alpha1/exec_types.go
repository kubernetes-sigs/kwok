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
	// ExecKind is the kind of the Exec.
	ExecKind = "Exec"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=execs,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=kwok.x-k8s.io,resources=execs/status,verbs=update;patch

// Exec provides exec configuration for a single pod.
type Exec struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata"`
	// Spec holds spec for exec
	Spec ExecSpec `json:"spec"`
	// Status holds status for exec
	//+k8s:conversion-gen=false
	Status ExecStatus `json:"status,omitempty"`
}

// ExecStatus holds status for exec
type ExecStatus struct {
	// Conditions holds conditions for exec
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ExecSpec holds spec for exec
type ExecSpec struct {
	// Execs is a list of execs to configure.
	Execs []ExecTarget `json:"execs"`
}

// ExecTarget holds information how to exec.
type ExecTarget struct {
	// Containers is a list of containers to exec.
	// if not set, all containers will be execed.
	Containers []string `json:"containers,omitempty"`
	// Local holds information how to exec to a local target.
	Local *ExecTargetLocal `json:"local,omitempty"`
	// Mapping is mapping to target
	Mapping *MappingTarget `json:"mapping,omitempty"`
}

// ExecTargetLocal holds information how to exec to a local target.
type ExecTargetLocal struct {
	// WorkDir is the working directory to exec with.
	WorkDir string `json:"workDir,omitempty"`
	// Envs is a list of environment variables to exec with.
	Envs []EnvVar `json:"envs,omitempty"`
	// SecurityContext is the user context to exec.
	SecurityContext *SecurityContext `json:"securityContext,omitempty"`
}

// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Value of the environment variable.
	Value string `json:"value,omitempty"`
}

// SecurityContext specifies the existing uid and gid to run exec command in container process.
type SecurityContext struct {
	// RunAsUser is the existing uid to run exec command in container process.
	RunAsUser *int64 `json:"runAsUser,omitempty"`
	// RunAsGroup is the existing gid to run exec command in container process.
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// ExecList contains a list of Exec
type ExecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Exec `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Exec{}, &ExecList{})
}
