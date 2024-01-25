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

package etcd

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

var patchMetaCache = map[schema.GroupVersionKind]strategicpatch.LookupPatchMeta{}

// PatchMetaFromStruct is a PatchMetaFromStruct implementation that uses strategicpatch.
func PatchMetaFromStruct(gvk schema.GroupVersionKind) (strategicpatch.LookupPatchMeta, error) {
	if obj, ok := patchMetaCache[gvk]; ok {
		return obj, nil
	}

	obj, err := Scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	pm, err := strategicpatch.NewPatchMetaFromStruct(obj)
	if err != nil {
		return nil, err
	}

	patchMetaCache[gvk] = pm

	return pm, nil
}
