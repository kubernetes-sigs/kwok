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

// ManagesSelectors holds information about the manages selectors.
type ManagesSelectors []ManagesSelector

// ManagesSelector holds information about the manages selector.
type ManagesSelector struct {
	// Kind of the referent.
	Kind string `json:"kind"`
	// Group of the referent.
	Group string `json:"group,omitempty"`
	// Version of the referent.
	Version string `json:"version,omitempty"`

	// Name of the referent
	// Only available with Node Kind.
	Name string `json:"name,omitempty"`
	// Labels of the referent.
	// specify matched with labels.
	// Only available with Node Kind.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations of the referent.
	// specify matched with annotations.
	// Only available with Node Kind.
	Annotations map[string]string `json:"annotations,omitempty"`
}
