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

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nowName         = "Now"
	mathRandName    = "Rand"
	sinceSecondName = "SinceSecond"
	unixSecondName  = "UnixSecond"

	quantityName = "Quantity"
)

var (
	DefaultTypes = []any{
		corev1.Node{},
		corev1.NodeSpec{},
		corev1.NodeStatus{},
		corev1.Pod{},
		corev1.PodSpec{},
		corev1.ResourceRequirements{},
		corev1.PodStatus{},
		corev1.Container{},
		metav1.ObjectMeta{},
		Quantity{},
		ResourceList{},
	}
	DefaultConversions = []any{
		func(t metav1.Time) types.Timestamp {
			return types.Timestamp{Time: t.Time}
		},
		func(t *metav1.Time) types.Timestamp {
			if t == nil {
				return types.Timestamp{}
			}
			return types.Timestamp{Time: t.Time}
		},
		func(t metav1.Duration) types.Duration {
			return types.Duration{Duration: t.Duration}
		},
		func(t *metav1.Duration) types.Duration {
			if t == nil {
				return types.Duration{}
			}
			return types.Duration{Duration: t.Duration}
		},
		func(t resource.Quantity) Quantity {
			return NewQuantity(&t)
		},
		NewResourceList,
	}
	DefaultFuncs = map[string][]any{
		nowName:         {timeNow},
		mathRandName:    {mathRand},
		sinceSecondName: {sinceSecond[*corev1.Node], sinceSecond[*corev1.Pod]},
		unixSecondName:  {unixSecond},
		quantityName:    {NewQuantityFromString},
	}
)

// FuncsToMethods converts a map of function names to functions to a map of method names to methods.
func FuncsToMethods(funcs map[string][]any) map[string][]any {
	methods := map[string][]any{}
	for name, funcs := range funcs {
		for _, fun := range funcs {
			t := reflect.TypeOf(fun)
			if t.Kind() != reflect.Func {
				continue
			}

			if t.NumIn() != 0 {
				methods[name] = append(methods[name], fun)
			}
		}
	}
	return methods
}

type (
	// Program is a cel program
	Program = cel.Program

	// Val is a cel value
	Val = ref.Val
)
