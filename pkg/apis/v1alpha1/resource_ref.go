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

// ResourceRef specifies the kind and version of the resource.
type ResourceRef struct {
	// APIGroup of the referent.
	// +default="v1"
	// +kubebuilder:default="v1"
	APIGroup string `json:"apiGroup,omitempty"`
	// Kind of the referent.
	Kind string `json:"kind"`
}
