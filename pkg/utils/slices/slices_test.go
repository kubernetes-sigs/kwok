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

package slices

import (
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	type args[S interface{ ~[]T }, T comparable] struct {
		s S
		t T
	}
	type testCase[S interface{ ~[]T }, T comparable] struct {
		name string
		args args[S, T]
		want bool
	}
	tests := []testCase[[]string, string]{
		{
			name: "test contains expect string",
			args: args[[]string, string]{
				s: []string{"a", "b", "c"},
				t: "b",
			},
			want: true,
		},
		{
			name: "test not contains expect string",
			args: args[[]string, string]{
				s: []string{"a", "b", "c"},
				t: "d",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.s, tt.args.t); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	type args[S interface{ ~[]T }, T any] struct {
		s S
		f func(T) bool
	}
	type testCase[S interface{ ~[]T }, T any] struct {
		name string
		args args[S, T]
		want []T
	}
	tests := []testCase[[]string, string]{
		{
			name: "test filter expect string",
			args: args[[]string, string]{
				s: []string{"a", "b", "c"},
				f: func(s string) bool {
					return s == "b"
				},
			},
			want: []string{"b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.args.s, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFind(t *testing.T) {
	type args[S interface{ ~[]T }, T any] struct {
		s S
		f func(T) bool
	}
	type testCase[S interface{ ~[]T }, T any] struct {
		name   string
		args   args[S, T]
		wantT  T
		wantOk bool
	}
	tests := []testCase[[]string, string]{
		{
			name: "test find expect string",
			args: args[[]string, string]{
				s: []string{"a", "b", "c"},
				f: func(s string) bool {
					return s == "b"
				},
			},
			wantT:  "b",
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotOk := Find(tt.args.s, tt.args.f)
			if !reflect.DeepEqual(gotT, tt.wantT) {
				t.Errorf("Find() gotT = %v, want %v", gotT, tt.wantT)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Find() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestMap(t *testing.T) {
	type args[S interface{ ~[]T }, T any, O any] struct {
		s S
		f func(T) O
	}
	type testCase[S interface{ ~[]T }, T any, O any] struct {
		name string
		args args[S, T, O]
		want []O
	}
	tests := []testCase[[]string, string, int]{
		{
			name: "test map expect string",
			args: args[[]string, string, int]{
				s: []string{"a", "b", "c"},
				f: func(s string) int {
					return len(s)
				},
			},
			want: []int{1, 1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Map(tt.args.s, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Map() = %v, want %v", got, tt.want)
			}
		})
	}
}
