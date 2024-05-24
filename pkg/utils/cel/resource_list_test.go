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
	"reflect"
	"testing"

	"github.com/google/cel-go/common/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewResourceList(t *testing.T) {
	list := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}
	rl := NewResourceList(list)

	if !reflect.DeepEqual(rl.List, list) {
		t.Errorf("expected %v, got %v", list, rl.List)
	}
}

func TestResourceList_ConvertToNative(t *testing.T) {
	list := corev1.ResourceList{}
	rl := NewResourceList(list)

	_, err := rl.ConvertToNative(reflect.TypeOf(""))
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestResourceList_ConvertToType(t *testing.T) {
	list := corev1.ResourceList{}
	rl := NewResourceList(list)

	val := rl.ConvertToType(types.StringType)
	if !types.IsError(val) {
		t.Errorf("expected error, got %v", val)
	}
}

func TestResourceList_Equal(t *testing.T) {
	list1 := corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("1"),
	}
	list2 := corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("1"),
	}
	list3 := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("2"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}

	rl1 := NewResourceList(list1)
	rl2 := NewResourceList(list2)
	rl3 := NewResourceList(list3)

	if !rl1.Equal(rl2).(types.Bool) {
		t.Errorf("expected lists to be equal")
	}

	if rl1.Equal(rl3).(types.Bool) {
		t.Errorf("expected lists to be not equal")
	}
}

func TestResourceList_Type(t *testing.T) {
	list := corev1.ResourceList{}
	rl := NewResourceList(list)

	if rl.Type() != ResourceListType {
		t.Errorf("expected %v, got %v", ResourceListType, rl.Type())
	}
}

func TestResourceList_Value(t *testing.T) {
	list := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}
	rl := NewResourceList(list)

	if !reflect.DeepEqual(rl.Value(), list) {
		t.Errorf("expected %v, got %v", list, rl.Value())
	}
}

func TestResourceList_Contains(t *testing.T) {
	list := corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("1"),
	}
	rl := NewResourceList(list)

	if !rl.Contains(types.String("cpu")).(types.Bool) {
		t.Errorf("expected resource list to contain 'cpu'")
	}

	if rl.Contains(types.String("memory")).(types.Bool) {
		t.Errorf("expected resource list to not contain 'memory'")
	}
}

func TestResourceList_Get(t *testing.T) {
	list := corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("1"),
	}
	rl := NewResourceList(list)
	cpuQuantity := list[corev1.ResourceCPU]

	if !rl.Get(types.String("cpu")).Equal(NewQuantity(&cpuQuantity)).(types.Bool) {
		t.Errorf("expected to get the value of 'cpu'")
	}

	if !rl.Get(types.String("memory")).Equal(NewQuantity(nil)).(types.Bool) {
		t.Errorf("expected to get nil for 'memory'")
	}
}

func TestResourceList_IsZeroValue(t *testing.T) {
	list := corev1.ResourceList{}
	rl := NewResourceList(list)

	if !rl.IsZeroValue() {
		t.Errorf("expected resource list to be zero value")
	}

	list = corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("1"),
	}
	rl = NewResourceList(list)

	if rl.IsZeroValue() {
		t.Errorf("expected resource list to not be zero value")
	}
}

func TestResourceList_Size(t *testing.T) {
	list := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	}
	rl := NewResourceList(list)

	if rl.Size() != types.Int(len(list)) {
		t.Errorf("expected size %d, got %v", len(list), rl.Size())
	}
}
