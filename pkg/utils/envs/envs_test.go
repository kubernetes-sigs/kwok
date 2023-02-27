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

package envs

import (
	"os"
	"reflect"
	"testing"
)

func TestGetEnv(t *testing.T) {
	type args[T any] struct {
		key string
		def T
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		preFunc func()
		want    T
	}
	tests := []testCase[string]{
		{
			name: "test get env not exist",
			args: args[string]{
				key: "test-key",
				def: "NOT-FOUND",
			},
			want: "NOT-FOUND",
		},
		{
			name: "test get env exist",
			args: args[string]{
				key: "test-key",
			},
			preFunc: func() {
				_ = os.Setenv("test-key", "test-value")
			},
			want: "test-value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preFunc != nil {
				tt.preFunc()
			}
			if got := GetEnv(tt.args.key, tt.args.def); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvWithPrefix(t *testing.T) {
	type args[T any] struct {
		key string
		def T
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		want    T
		preFunc func()
	}
	tests := []testCase[string]{
		{
			name: "test get env with prefix not exist",
			args: args[string]{
				key: EnvPrefix + "test-key",
				def: "NOT-FOUND",
			},
			want: "NOT-FOUND",
		},
		{
			name: "test get env with prefix exist",
			args: args[string]{
				key: "test-key",
			},
			preFunc: func() {
				_ = os.Setenv(EnvPrefix+"test-key", "test-value")
			},
			want: "test-value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preFunc != nil {
				tt.preFunc()
			}
			if got := GetEnvWithPrefix(tt.args.key, tt.args.def); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEnvWithPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
