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

package config

import (
	"reflect"
	"testing"
)

func Test_loadConfig(t *testing.T) {
	type args struct {
		configPaths        []string
		defaultConfigPath  string
		existDefaultConfig bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "default config",
			args: args{
				configPaths:        []string{},
				defaultConfigPath:  "default",
				existDefaultConfig: true,
			},
			want: []string{"default"},
		},
		{
			name: "add default config",
			args: args{
				configPaths:        []string{"config"},
				defaultConfigPath:  "default",
				existDefaultConfig: true,
			},
			want: []string{"default", "config"},
		},
		{
			name: "no default config",
			args: args{
				configPaths:        []string{"config"},
				defaultConfigPath:  "default",
				existDefaultConfig: false,
			},
			want: []string{"config"},
		},
		{
			name: "no config",
			args: args{
				configPaths:        []string{},
				defaultConfigPath:  "default",
				existDefaultConfig: false,
			},
			want: []string{},
		},
		{
			name: "remove default config",
			args: args{
				configPaths:        []string{"default", "config"},
				defaultConfigPath:  "default",
				existDefaultConfig: false,
			},
			want: []string{"config"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loadConfig(tt.args.configPaths, tt.args.defaultConfigPath, tt.args.existDefaultConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
