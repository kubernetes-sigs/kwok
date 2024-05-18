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

package patch

import (
	"reflect"
	"testing"
)

type testStruct struct {
	A string           `json:"a,omitempty"`
	B int              `json:"b,omitempty"`
	C []testStructItem `json:"c,omitempty" patchStrategy:"merge" patchMergeKey:"a"`
}

type testStructItem struct {
	A string `json:"a,omitempty"`
	B int    `json:"b,omitempty"`
}

func TestStrategicMerge(t *testing.T) {
	type args[T any] struct {
		original T
		patch    T
	}
	type testCase[T any] struct {
		name       string
		args       args[T]
		wantResult T
		wantErr    bool
	}
	tests := []testCase[testStruct]{
		{
			name: "test1",
			args: args[testStruct]{
				original: testStruct{
					A: "a",
				},
				patch: testStruct{
					B: 1,
				},
			},
			wantResult: testStruct{
				A: "a",
				B: 1,
			},
		},
		{
			name: "test2",
			args: args[testStruct]{
				original: testStruct{
					A: "a",
					C: []testStructItem{
						{
							A: "a",
						},
					},
				},
				patch: testStruct{
					B: 1,
					C: []testStructItem{
						{
							A: "b",
							B: 1,
						},
					},
				},
			},
			wantResult: testStruct{
				A: "a",
				B: 1,
				C: []testStructItem{
					{
						A: "b",
						B: 1,
					},
					{
						A: "a",
					},
				},
			},
		},
		{
			name: "test3",
			args: args[testStruct]{
				original: testStruct{
					A: "a",
					C: []testStructItem{
						{
							A: "a",
						},
					},
				},
				patch: testStruct{
					B: 1,
					C: []testStructItem{
						{
							A: "b",
							B: 1,
						},
						{
							A: "a",
						},
					},
				},
			},
			wantResult: testStruct{
				A: "a",
				B: 1,
				C: []testStructItem{
					{
						A: "b",
						B: 1,
					},
					{
						A: "a",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := StrategicMerge(tt.args.original, tt.args.patch)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrategicMerge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("StrategicMerge() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}

func TestStrategicMergePatch(t *testing.T) {
	type args[T any] struct {
		original  T
		patchData []byte
	}
	type testCase[T any] struct {
		name       string
		args       args[T]
		wantResult T
		wantErr    bool
	}
	tests := []testCase[testStruct]{
		{
			name: "test1",
			args: args[testStruct]{
				original: testStruct{
					A: "a",
				},
				patchData: []byte(`{"B":1}`),
			},
			wantResult: testStruct{
				A: "a",
				B: 1,
			},
		},
		{
			name: "test2",
			args: args[testStruct]{
				original: testStruct{
					A: "b",
					B: 2,
				},
				patchData: []byte(`{"C":[{"A":"c","B":3}]}`),
			},
			wantResult: testStruct{
				A: "b",
				B: 2,
				C: []testStructItem{
					{
						A: "c",
						B: 3,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := StrategicMergePatch(tt.args.original, tt.args.patchData)
			if (err != nil) != tt.wantErr {
				t.Errorf("StrategicMergePatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotResult, tt.wantResult) {
				t.Errorf("StrategicMergePatch() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
