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

package sets

import (
	"reflect"
	"testing"
)

// TestNewSets verifies that NewSets creates a set with the given items.
func TestNewSets(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  Sets[int]
	}{
		{
			name:  "Empty set",
			items: []int{},
			want:  Sets[int]{},
		},
		{
			name:  "Single item",
			items: []int{1},
			want:  Sets[int]{1: {}},
		},
		{
			name:  "Multiple items",
			items: []int{1, 2, 3},
			want:  Sets[int]{1: {}, 2: {}, 3: {}},
		},
		{
			name:  "Duplicate items",
			items: []int{1, 2, 2, 3, 1},
			want:  Sets[int]{1: {}, 2: {}, 3: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSets(tt.items...)
			if !compareSets(got, tt.want) {
				t.Errorf("NewSets() = %v, want %v", got, tt.want)
			}
		})
	}
}

// compareSets compares two Sets for equality.
func compareSets[T comparable](a, b Sets[T]) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  Sets[int]
	}{
		{
			name:  "Single item",
			items: []int{1},
			want:  Sets[int]{1: {}},
		},
		{
			name:  "Multiple items",
			items: []int{1, 2, 3},
			want:  Sets[int]{1: {}, 2: {}, 3: {}},
		},
		{
			name:  "Duplicate items",
			items: []int{2, 2, 3},
			want:  Sets[int]{2: {}, 3: {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := make(Sets[int])
			set.Insert(tt.items...)

			if !reflect.DeepEqual(set, tt.want) {
				t.Errorf("Insert(%v): want %v, got %v", tt.items, tt.want, set)
			}
		})
	}
}

func TestEmptySet(t *testing.T) {
	set := make(Sets[int])

	wantLen := 0
	gotLen := len(set)
	if gotLen != wantLen {
		t.Errorf("Initial set: want length to be %d, got %d", wantLen, gotLen)
	}
}

func TestInsertString(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  Sets[string]
	}{
		{
			name:  "Single string item",
			items: []string{"a"},
			want:  Sets[string]{"a": {}},
		},
		{
			name:  "Multiple string items",
			items: []string{"b", "c", "d"},
			want:  Sets[string]{"b": {}, "c": {}, "d": {}},
		},
		{
			name:  "Duplicate string items",
			items: []string{"d", "e", "e"},
			want:  Sets[string]{"d": {}, "e": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := make(Sets[string])
			set.Insert(tt.items...)

			if !reflect.DeepEqual(set, tt.want) {
				t.Errorf("Insert(%v): want %v, got %v", tt.items, tt.want, set)
			}
		})
	}
}

func TestSets_Has(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  Sets[string]
	}{
		{
			name:  "Test empty set",
			items: []string{},
			want:  Sets[string]{},
		},
		{
			name:  "Test set with single item",
			items: []string{"apple"},
			want:  Sets[string]{"apple": struct{}{}},
		},
		{
			name:  "Test set with multiple items",
			items: []string{"apple", "banana", "orange"},
			want:  Sets[string]{"apple": struct{}{}, "banana": struct{}{}, "orange": struct{}{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make(Sets[string])
			for _, item := range tt.items {
				got[item] = struct{}{}
			}

			for item := range got {
				if !tt.want.Has(item) {
					t.Errorf("Sets.Has() = false, want true for item %v", item)
				}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sets.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSets_Delete(t *testing.T) {
	tests := []struct {
		name       string
		initialSet Sets[string]
		items      []string
		want       Sets[string]
	}{
		{
			name:       "Delete single item",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
			items:      []string{"a"},
			want:       Sets[string]{"b": {}, "c": {}}, // Expecting set after deleting "a"
		},
		{
			name:       "Delete multiple items",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
			items:      []string{"a", "b"},
			want:       Sets[string]{"c": {}}, // Expecting set after deleting "a" and "b"
		},
		{
			name:       "Delete non-existent item",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
			items:      []string{"d"},
			want:       Sets[string]{"a": {}, "b": {}, "c": {}}, // Expecting no change as "d" does not exist
		},
		{
			name:       "Delete all items",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
			items:      []string{"a", "b", "c"},
			want:       Sets[string]{}, // Expecting an empty set after deleting all items
		},
		{
			name:       "Delete from empty set",
			initialSet: Sets[string]{},
			items:      []string{"a", "b", "c"},
			want:       Sets[string]{}, // Expecting no change as set is empty
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.initialSet
			s.Delete(tc.items...)
			if !reflect.DeepEqual(s, tc.want) {
				t.Errorf("got %v; want %v", s, tc.want)
			}
		})
	}

	// Test for deleting from an empty set
	t.Run("Delete from empty set", func(t *testing.T) {
		emptySet := Sets[string]{} // Empty set
		emptySet.Delete("a")
		if !reflect.DeepEqual(emptySet, Sets[string]{}) {
			t.Errorf("got %v; want %v", emptySet, Sets[string]{})
		}
	})
}

func TestSets_Len(t *testing.T) {
	tests := []struct {
		name       string
		initialSet Sets[string]
		want       int
	}{
		{
			name:       "Empty set",
			initialSet: Sets[string]{},
			want:       0,
		},
		{
			name:       "Set with elements",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
			want:       3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.initialSet.Len(); got != tc.want {
				t.Errorf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestSets_Clear(t *testing.T) {
	tests := []struct {
		name       string
		initialSet Sets[string]
	}{
		{
			name:       "Empty set",
			initialSet: Sets[string]{},
		},
		{
			name:       "Set with elements",
			initialSet: Sets[string]{"a": {}, "b": {}, "c": {}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.initialSet.Clear()
			if len(tc.initialSet) != 0 {
				t.Errorf("Expected the set to be empty after Clear(), but it still has %d items", len(tc.initialSet))
			}
		})
	}
}
