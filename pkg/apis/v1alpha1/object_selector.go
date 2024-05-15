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

// ObjectSelector holds information on how to match based on namespace and name.
type ObjectSelector struct {
	// MatchNamespaces is a list of namespaces to match.
	// if not set, all namespaces will be matched.
	MatchNamespaces []string `json:"matchNamespaces,omitempty"`
	// MatchNames is a list of names to match.
	// if not set, all names will be matched.
	MatchNames []string `json:"matchNames,omitempty"`
}
