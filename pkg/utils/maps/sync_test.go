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

package maps

import (
	"reflect"
	"testing"
)

func TestSyncMap_Delete(t *testing.T) {
	type args[K comparable] struct {
		key K
	}
	type testCase[K comparable, V any] struct {
		name         string
		args         args[K]
		createFunc   func() *SyncMap[K, V]
		validateFunc func(m *SyncMap[K, V]) bool
	}
	tests := []testCase[string, string]{
		{
			name: "test delete exists key",

			args: args[string]{
				key: "key",
			},
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.m.Store("key", "value")
				return s
			},
			validateFunc: func(m *SyncMap[string, string]) bool {
				_, ok := m.Load("key")
				return !ok
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.createFunc()
			s.Delete(tt.args.key)
			if tt.validateFunc != nil {
				if !tt.validateFunc(s) {
					t.Errorf("Delete() validateFunc = false, want true")
				}
			}
		})
	}
}

func TestSyncMap_Load(t *testing.T) {
	type args[K comparable] struct {
		key K
	}
	type testCase[K comparable, V any] struct {
		name       string
		args       args[K]
		wantValue  V
		wantOk     bool
		createFunc func() *SyncMap[K, V]
	}
	tests := []testCase[string, string]{
		{
			name: "test exists key",
			args: args[string]{
				key: "key",
			},
			wantValue: "value",
			wantOk:    true,
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.m.Store("key", "value")
				return s
			},
		},
		{
			name: "test not exists key",
			args: args[string]{
				key: "key",
			},
			wantValue: "",
			wantOk:    false,
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				return s
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			gotValue, gotOk := m.Load(tt.args.key)
			if !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("Load() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Load() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestSyncMap_LoadAndDelete(t *testing.T) {
	type args[K comparable] struct {
		key K
	}
	type testCase[K comparable, V any] struct {
		name       string
		args       args[K]
		wantValue  V
		createFunc func() *SyncMap[K, V]
		validFunc  func(m *SyncMap[K, V]) bool
		wantLoaded bool
	}
	tests := []testCase[string, string]{
		{
			name: "test exists key",
			args: args[string]{
				key: "key",
			},
			wantValue: "value",
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.Store("key", "value")
				return s
			},
			validFunc: func(m *SyncMap[string, string]) bool {
				_, ok := m.Load("key")
				return !ok
			},
			wantLoaded: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			gotValue, gotLoaded := m.LoadAndDelete(tt.args.key)
			if !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("LoadAndDelete() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
			if gotLoaded != tt.wantLoaded {
				t.Errorf("LoadAndDelete() gotLoaded = %v, want %v", gotLoaded, tt.wantLoaded)
			}
			if tt.validFunc != nil {
				if !tt.validFunc(m) {
					t.Errorf("LoadAndDelete() validFunc = false, want true")
				}
			}
		})
	}
}

func TestSyncMap_LoadOrStore(t *testing.T) {
	type args[K comparable, V any] struct {
		key   K
		value V
	}
	type testCase[K comparable, V any] struct {
		name       string
		args       args[K, V]
		wantActual V
		createFunc func() *SyncMap[K, V]
		wantLoaded bool
	}
	tests := []testCase[string, string]{
		{
			name: "test exists key",
			args: args[string, string]{
				key:   "key",
				value: "value",
			},
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.Store("key", "value")
				return s
			},
			wantActual: "value",
			wantLoaded: true,
		},
		{
			name: "test not exists key",
			args: args[string, string]{
				key:   "key",
				value: "value",
			},
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				return s
			},
			wantActual: "value",
			wantLoaded: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			gotActual, gotLoaded := m.LoadOrStore(tt.args.key, tt.args.value)
			if !reflect.DeepEqual(gotActual, tt.wantActual) {
				t.Errorf("LoadOrStore() gotActual = %v, want %v", gotActual, tt.wantActual)
			}
			if gotLoaded != tt.wantLoaded {
				t.Errorf("LoadOrStore() gotLoaded = %v, want %v", gotLoaded, tt.wantLoaded)
			}
		})
	}
}

func TestSyncMap_Range(t *testing.T) {
	type args[K comparable, V any] struct {
		f func(key K, value V) bool
	}
	type testCase[K comparable, V any] struct {
		name         string
		args         args[K, V]
		want         string
		createFunc   func() *SyncMap[K, V]
		validateFunc func(s string) bool
	}
	tests := []testCase[string, string]{
		{
			name: "test range func",
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.Store("test", "test")
				return s
			},
			args: args[string, string]{
				f: func(key string, value string) bool {
					return key == "test"
				},
			},
			want: "test",
			validateFunc: func(s string) bool {
				return s == "test"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			m.Range(tt.args.f)
			if tt.validateFunc != nil {
				if !tt.validateFunc(tt.want) {
					t.Errorf("Range() validateFunc = false, want true")
				}
			}
		})
	}
}

func TestSyncMap_Size(t *testing.T) {
	type testCase[K comparable, V any] struct {
		name       string
		want       int
		createFunc func() *SyncMap[K, V]
	}
	tests := []testCase[string, string]{
		{
			name: "test empty",
			createFunc: func() *SyncMap[string, string] {
				return &SyncMap[string, string]{}
			},
			want: 0,
		},
		{
			name: "test get map size",
			createFunc: func() *SyncMap[string, string] {
				s := &SyncMap[string, string]{}
				s.Store("key", "value")
				s.Store("key1", "value1")
				return s
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			if got := m.Size(); got != tt.want {
				t.Errorf("Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncMap_Store(t *testing.T) {
	type args[K comparable, V any] struct {
		key   K
		value V
	}
	type testCase[K comparable, V any] struct {
		name         string
		args         args[K, V]
		createFunc   func() *SyncMap[K, V]
		validateFunc func(m *SyncMap[K, V]) bool
	}
	tests := []testCase[string, string]{
		{
			name: "test store success",
			args: args[string, string]{
				key:   "key",
				value: "value",
			},
			createFunc: func() *SyncMap[string, string] {
				return &SyncMap[string, string]{}
			},
			validateFunc: func(m *SyncMap[string, string]) bool {
				v, ok := m.m.Load("key")
				if !ok {
					return false
				}
				return v == "value"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.createFunc()
			m.Store(tt.args.key, tt.args.value)
			if tt.validateFunc != nil {
				if !tt.validateFunc(m) {
					t.Errorf("Store() validateFunc = false, want true")
				}
			}
		})
	}
}
