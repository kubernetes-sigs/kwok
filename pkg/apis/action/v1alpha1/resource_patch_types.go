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
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourcePatchKind is the kind of the ResourcePatch.
	ResourcePatchKind = "ResourcePatch"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourcePatch provides resource definition for kwokctl.
// this is a action of resource patch.
type ResourcePatch struct {
	//+k8s:conversion-gen=false
	metav1.TypeMeta `json:",inline"`

	// Resource represents the resource to be patched.
	Resource GroupVersionResource `json:"resource"`
	// Target represents the target of the ResourcePatch.
	Target Target `json:"target"`
	// DurationNanosecond represents the duration of the patch in nanoseconds.
	DurationNanosecond time.Duration `json:"durationNanosecond"`
	// Method represents the method of the patch.
	Method PatchMethod `json:"method"`
	// Template contains the patch data as a raw JSON message.
	Template json.RawMessage `json:"template,omitempty"`
}

type PatchMethod string

const (
	// PatchMethodCreate means that the resource will be created by create.
	PatchMethodCreate PatchMethod = "create"
	// PatchMethodPatch means that the resource will be patched by patch.
	PatchMethodPatch PatchMethod = "patch"
	// PatchMethodDelete means that the resource will be deleted by delete.
	PatchMethodDelete PatchMethod = "delete"
)

// Target is a struct that represents the target of the ResourcePatch.
type Target struct {
	// Name represents the name of the resource to be patched.
	Name string `json:"name"`
	// Namespace represents the namespace of the resource to be patched.
	Namespace string `json:"namespace,omitempty"`
}

// GroupVersionResource is a struct that represents the group version resource.
type GroupVersionResource struct {
	// Group represents the group of the resource.
	Group string `json:"group,omitempty"`
	// Version represents the version of the resource.
	Version string `json:"version"`
	// Resource represents the type of the resource.
	Resource string `json:"resource"`
}
