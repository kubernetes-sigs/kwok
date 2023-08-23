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

package version

import (
	"testing"
)

func TestVersionInfo(t *testing.T) {
	type args struct {
		version    string
		preRelease string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "versionInfo to alpha",
			args: args{
				version:    "v1.2.3",
				preRelease: "alpha",
			},
			want: "1.2.3-alpha",
		},
		{
			name: "versionInfo to alpha but version is not v",
			args: args{
				version:    "1.2.3",
				preRelease: "alpha",
			},
			want: "1.2.3-alpha",
		},
		{
			name: "versionInfo to alpha but version is empty",
			args: args{
				version:    "",
				preRelease: "alpha",
			},
			want: "",
		},
		{
			name: "versionInfo to GA",
			args: args{
				version:    "v1.2.3",
				preRelease: "GA",
			},
			want: "1.2.3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionInfo(tt.args.version, tt.args.preRelease); got != tt.want {
				t.Errorf("VersionInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
