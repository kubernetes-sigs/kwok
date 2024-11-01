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
	"time"

	"github.com/google/cel-go/common/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFuncsToMethods(t *testing.T) {
	funcs := map[string][]any{
		nowName:         {timeNow},
		mathRandName:    {mathRand},
		sinceSecondName: {sinceSecond[*corev1.Node], sinceSecond[*corev1.Pod]},
		unixSecondName:  {unixSecond},
		quantityName:    {NewQuantityFromString},
	}

	methods := FuncsToMethods(funcs)

	expectedMethods := map[string][]any{
		sinceSecondName: {sinceSecond[*corev1.Node], sinceSecond[*corev1.Pod]},
	}

	for name, funcs := range expectedMethods {
		if _, ok := methods[name]; !ok {
			t.Errorf("Expected method %s not found in the result", name)
			continue
		}

		for i, fun := range funcs {
			if reflect.TypeOf(methods[name][i]) != reflect.TypeOf(fun) {
				t.Errorf("Expected method %s of type %v, but got %v", name, reflect.TypeOf(fun), reflect.TypeOf(methods[name][i]))
			}
		}
	}
}

func TestDefaultConversions(t *testing.T) {
	timeValue := metav1.Time{Time: time.Now()}
	durationValue := metav1.Duration{Duration: time.Second}
	quantityValue := resource.MustParse("1Gi")

	tests := []struct {
		name string
		fn   any
		arg  any
		want any
	}{
		{
			name: "metav1.Time to types.Timestamp",
			fn:   DefaultConversions[0],
			arg:  timeValue,
			want: types.Timestamp{Time: timeValue.Time},
		},
		{
			name: "*metav1.Time to types.Timestamp",
			fn:   DefaultConversions[1],
			arg:  &timeValue,
			want: types.Timestamp{Time: timeValue.Time},
		},
		{
			name: "metav1.Duration to types.Duration",
			fn:   DefaultConversions[2],
			arg:  durationValue,
			want: types.Duration{Duration: durationValue.Duration},
		},
		{
			name: "*metav1.Duration to types.Duration",
			fn:   DefaultConversions[3],
			arg:  &durationValue,
			want: types.Duration{Duration: durationValue.Duration},
		},
		{
			name: "resource.Quantity to Quantity",
			fn:   DefaultConversions[4],
			arg:  quantityValue,
			want: NewQuantity(&quantityValue),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fnValue := reflect.ValueOf(tt.fn)
			argValue := reflect.ValueOf(tt.arg)
			result := fnValue.Call([]reflect.Value{argValue})[0].Interface()
			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("Expected %v, but got %v", tt.want, result)
			}
		})
	}
}
