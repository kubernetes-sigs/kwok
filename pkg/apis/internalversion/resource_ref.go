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

package internalversion

// ResourceRef specifies the kind and version of the resource.
type ResourceRef struct {
	// APIGroup of the referent.
	APIGroup string
	// Kind of the referent.
	Kind string
}

// Match returns true if apiGroup and kind is specified within the selector
// If the match field is empty, the match on that field is considered to be true.
func (r *ResourceRef) Match(apiGroup, kind string) bool {
	if r == nil {
		return true
	}
	if r.APIGroup != apiGroup {
		return false
	}
	if r.Kind != kind {
		return false
	}
	return true
}
