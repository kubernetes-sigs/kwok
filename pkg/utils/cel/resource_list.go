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

package cel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	corev1 "k8s.io/api/core/v1"
)

var (
	// ResourceListType singleton.
	ResourceListType = cel.ObjectType("kubernetes.ResourceList",
		traits.ContainerType,
		traits.IndexerType,
		traits.SizerType,
	)
)

// ResourceList is a wrapper around k8s.io/api/core/v1.ResourceList
type ResourceList struct {
	List corev1.ResourceList
}

// NewResourceList creates a new ResourceList
func NewResourceList(list corev1.ResourceList) ResourceList {
	return ResourceList{List: list}
}

// ConvertToNative implements the ref.Val interface.
func (r ResourceList) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("unsupported conversion from '%s' to '%v'", ResourceListType, typeDesc)
}

// ConvertToType implements the ref.Val interface.
func (r ResourceList) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("type conversion error from '%s' to '%s'", ResourceListType, typeValue)
}

// Equal implements the ref.Val interface.
func (r ResourceList) Equal(other ref.Val) ref.Val {
	otherResourceList, ok := other.(ResourceList)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	if len(r.List) != len(otherResourceList.List) {
		return types.False
	}
	for k, v := range r.List {
		otherVal, ok := otherResourceList.List[k]
		if !ok {
			return types.False
		}
		if !v.Equal(otherVal) {
			return types.False
		}
	}
	return types.True
}

// Type implements the ref.Val interface.
func (r ResourceList) Type() ref.Type {
	return ResourceListType
}

// Value implements the ref.Val interface.
func (r ResourceList) Value() any {
	return r.List
}

// Contains implements the traits.Container interface.
func (r ResourceList) Contains(index ref.Val) ref.Val {
	key, ok := index.(types.String)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}
	_, found := r.List[corev1.ResourceName(key)]
	return types.Bool(found)
}

// Get implements the traits.Indexer interface.
func (r ResourceList) Get(index ref.Val) ref.Val {
	key, ok := index.(types.String)
	if !ok {
		return types.MaybeNoSuchOverloadErr(index)
	}

	val, found := r.List[corev1.ResourceName(key)]
	if !found {
		return NewQuantity(nil)
	}
	return NewQuantity(&val)
}

// IsZeroValue implements the traits.Zeroer interface.
func (r ResourceList) IsZeroValue() bool {
	return len(r.List) == 0
}

// Size implements the traits.Sizer interface.
func (r ResourceList) Size() ref.Val {
	return types.Int(len(r.List))
}
