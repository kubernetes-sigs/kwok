package maps

import (
	"reflect"
	"testing"
)

func TestKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want []string
	}{
		{
			name: "Empty map",
			m:    map[string]int{},
			want: []string{},
		},
		{
			name: "Single key",
			m:    map[string]int{"a": 1},
			want: []string{"a"},
		},
		{
			name: "Multiple keys",
			m:    map[string]int{"a": 1, "b": 2, "c": 3},
			want: []string{"a", "b", "c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Keys(tc.m)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Keys() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValues(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]int
		want []int
	}{
		{
			name: "Empty map",
			m:    map[string]int{},
			want: []int{},
		},
		{
			name: "Single value",
			m:    map[string]int{"a": 1},
			want: []int{1},
		},
		{
			name: "Multiple values",
			m:    map[string]int{"a": 1, "b": 2, "c": 3},
			want: []int{1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Values(tc.m)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Values() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		maps []map[string]int
		want map[string]int
	}{
		{
			name: "No maps",
			maps: nil,
			want: nil,
		},
		{
			name: "Single map",
			maps: []map[string]int{
				{"a": 1, "b": 2},
			},
			want: map[string]int{"a": 1, "b": 2},
		},
		{
			name: "Multiple maps with no conflicts",
			maps: []map[string]int{
				{"a": 1},
				{"b": 2},
				{"c": 3},
			},
			want: map[string]int{"a": 1, "b": 2, "c": 3},
		},
		{
			name: "Multiple maps with conflicts",
			maps: []map[string]int{
				{"a": 1},
				{"a": 2, "b": 2},
				{"a": 3, "b": 3, "c": 3},
			},
			want: map[string]int{"a": 3, "b": 3, "c": 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Merge(tc.maps...)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Merge() = %v, want %v", got, tc.want)
			}
		})
	}
}
