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
	"sync"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/wzshiming/easycel"
)

func TestNewEnvironment(t *testing.T) {
	type args struct {
		conf EnvironmentConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *Environment
		wantErr bool
	}{
        args: args{
			&EnvironmentConfig{
			Conversions: []any{},
			Types:       []any{},
			 Vars:        map[string]any{},
			Funcs:       map[string][]any{},
			Methods:     map[string][]any{},
		},     
		
},		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEnvironment(tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironment_Compile(t *testing.T) {
	type fields struct {
		env          *easycel.Environment
		cacheProgram map[string]cel.Program
		cacheMut     sync.Mutex
	}
	type args struct {
		src string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    cel.Program
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Environment{
				env:          tt.fields.env,
				cacheProgram: tt.fields.cacheProgram,
				cacheMut:     tt.fields.cacheMut,
			}
			got, err := e.Compile(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Environment.Compile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Environment.Compile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsFloat64(t *testing.T) {
	type args struct {
		refVal ref.Val
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AsFloat64(tt.args.refVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsString(t *testing.T) {
	type args struct {
		refVal ref.Val
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AsString(tt.args.refVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("AsString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AsString() = %v, want %v", got, tt.want)
			}
		})
	}
}
