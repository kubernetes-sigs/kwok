/*
Copyright 2022 The Kubernetes Authors.

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

package format

import (
	"reflect"
	"testing"
)

func TestParseString(t *testing.T) {
	type args struct {
		s string
	}
	type testCase[T any] struct {
		name    string
		args    args
		wantT   T
		wantErr bool
	}
	tests := []testCase[string]{
		{
			args: args{
				s: "hello",
			},
			wantT:   "hello",
			wantErr: false,
		},
		{
			args: args{
				s: `"hello"`,
			},
			wantT:   `"hello"`,
			wantErr: false,
		},
		{
			args: args{
				s: "hello world",
			},
			wantT:   "hello",
			wantErr: false,
		},
		{
			args: args{
				s: " hello",
			},
			wantT:   "hello",
			wantErr: false,
		},
		{
			args: args{
				s: "https://localhost:8080/xxx",
			},
			wantT:   "https://localhost:8080/xxx",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, err := Parse[string](tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotT, tt.wantT) {
				t.Errorf("Parse() gotT = %v, want %v", gotT, tt.wantT)
			}
		})
	}
}
