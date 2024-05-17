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
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	type args[S ~[]T, T comparable] struct {
		s S
		t T
	}
	type testCase[S ~[]T, T comparable] struct {
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
	type args[S ~[]T, T any] struct {
		s S
		f func(T) bool
	}
	type testCase[S ~[]T, T any] struct {
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
	type args[S ~[]T, T any] struct {
		s S
		f func(T) bool
	}
	type testCase[S ~[]T, T any] struct {
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
	type args[S ~[]T, T any, O any] struct {
		s S
		f func(T) O
	}
	type testCase[S ~[]T, T any, O any] struct {
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

func TestMapWithError(t *testing.T) {
	type args[S ~[]T, T any, O any] struct {
		s S
		f func(T) (O, error)
	}
	type testCase[S ~[]T, T any, O any] struct {
		name    string
		args    args[S, T, O]
		want    []O
		wantErr bool
	}
	tests := []testCase[[]string, string, int]{
		{
			name: "test map with error no error",
			args: args[[]string, string, int]{
				s: []string{"a", "b", "c"},
				f: func(s string) (int, error) {
					return len(s), nil
				},
			},
			want:    []int{1, 1, 1},
			wantErr: false,
		},
		{
			name: "test map with error with error",
			args: args[[]string, string, int]{
				s: []string{"a", "b", "c"},
				f: func(s string) (int, error) {
					if s == "b" {
						return 0, errors.New("error")
					}
					return len(s), nil
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "test map with error empty slice",
			args: args[[]string, string, int]{
				s: []string{},
				f: func(s string) (int, error) {
					return len(s), nil
				},
			},
			want:    []int{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapWithError(tt.args.s, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("MapWithError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapWithError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterAndMap(t *testing.T) {
	type args[S ~[]T, T any, O any] struct {
		s S
		f func(T) (O, bool)
	}
	type testCase[S ~[]T, T any, O any] struct {
		name string
		args args[S, T, O]
		want []O
	}
	tests := []testCase[[]string, string, int]{
		{
			name: "test filter and map",
			args: args[[]string, string, int]{
				s: []string{"a", "b", "c"},
				f: func(s string) (int, bool) {
					if s == "b" {
						return 2, true
					}
					return 0, false
				},
			},
			want: []int{2},
		},
		{
			name: "test filter and map no match",
			args: args[[]string, string, int]{
				s: []string{"a", "b", "c"},
				f: func(s string) (int, bool) {
					return 0, false
				},
			},
			want: []int{},
		},
		{
			name: "test filter and map empty slice",
			args: args[[]string, string, int]{
				s: []string{},
				f: func(s string) (int, bool) {
					return 0, false
				},
			},
			want: []int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterAndMap(tt.args.s, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterAndMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	type args[S ~[]T, T comparable] struct {
		s S
	}
	type testCase[S ~[]T, T comparable] struct {
		name string
		args args[S, T]
		want []T
	}
	tests := []testCase[[]int, int]{
		{
			name: "test unique with duplicates",
			args: args[[]int, int]{
				s: []int{1, 2, 2, 3, 4, 4, 4, 5},
			},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "test unique without duplicates",
			args: args[[]int, int]{
				s: []int{1, 2, 3, 4, 5},
			},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			name: "test unique empty slice",
			args: args[[]int, int]{
				s: []int{},
			},
			want: []int{},
		},
		{
			name: "test unique single element",
			args: args[[]int, int]{
				s: []int{1},
			},
			want: []int{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unique(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	type args[S ~[]T, T comparable] struct {
		s1 S
		s2 S
	}
	type testCase[S ~[]T, T comparable] struct {
		name string
		args args[S, T]
		want bool
	}
	tests := []testCase[[]int, int]{
		{
			name: "test equal slices",
			args: args[[]int, int]{
				s1: []int{1, 2, 3, 4, 5},
				s2: []int{1, 2, 3, 4, 5},
			},
			want: true,
		},
		{
			name: "test not equal slices different lengths",
			args: args[[]int, int]{
				s1: []int{1, 2, 3, 4, 5},
				s2: []int{1, 2, 3, 4},
			},
			want: false,
		},
		{
			name: "test not equal slices different elements",
			args: args[[]int, int]{
				s1: []int{1, 2, 3, 4, 5},
				s2: []int{1, 2, 3, 4, 6},
			},
			want: false,
		},
		{
			name: "test equal empty slices",
			args: args[[]int, int]{
				s1: []int{},
				s2: []int{},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Equal(tt.args.s1, tt.args.s2); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	type args[S ~[]T, T any] struct {
		s S
	}
	type testCase[S ~[]T, T any] struct {
		name string
		args args[S, T]
		want []T
	}
	tests := []testCase[[]int, int]{
		{
			name: "test reverse",
			args: args[[]int, int]{
				s: []int{1, 2, 3, 4, 5},
			},
			want: []int{5, 4, 3, 2, 1},
		},
		{
			name: "test reverse empty slice",
			args: args[[]int, int]{
				s: []int{},
			},
			want: []int{},
		},
		{
			name: "test reverse single element",
			args: args[[]int, int]{
				s: []int{1},
			},
			want: []int{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reverse(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reverse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupBy(t *testing.T) {
	type args[S ~[]T, T any, K comparable] struct {
		s S
		f func(T) K
	}
	type testCase[S ~[]T, T any, K comparable] struct {
		name string
		args args[S, T, K]
		want map[K][]T
	}
	tests := []testCase[[]string, string, string]{
		{
			name: "test group by first letter",
			args: args[[]string, string, string]{
				s: []string{"apple", "apricot", "banana", "cherry"},
				f: func(s string) string {
					return string(s[0])
				},
			},
			want: map[string][]string{
				"a": {"apple", "apricot"},
				"b": {"banana"},
				"c": {"cherry"},
			},
		},
		{
			name: "test group by length",
			args: args[[]string, string, string]{
				s: []string{"apple", "banana", "cherry", "date"},
				f: func(s string) string {
					return fmt.Sprintf("%d", len(s))
				},
			},
			want: map[string][]string{
				"5": {"apple"},
				"6": {"banana", "cherry"},
				"4": {"date"},
			},
		},
		{
			name: "test group by empty slice",
			args: args[[]string, string, string]{
				s: []string{},
				f: func(s string) string {
					return fmt.Sprintf("%d", len(s))
				},
			},
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GroupBy(tt.args.s, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupBy() = %v, want %v", got, tt.want)
			}
		})
	}
}
