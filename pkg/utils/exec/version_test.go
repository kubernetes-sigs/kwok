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

package exec

import (
	"context"
	"reflect"
	"testing"

	"github.com/blang/semver/v4"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

func TestParseVersionFromBinary(t *testing.T) {
	type args struct {
		ctx  context.Context
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    version.Version
		wantErr bool
	}{{
		name: "testing go",
		args: args{
			ctx:  context.Background(),
			path: "go",
		},
		want:    semver.MustParse("1.22.3"),
		wantErr: false,
	},
		{
			name: "testing kind",
			args: args{
				ctx:  context.Background(),
				path: "kind",
			},
			want: semver.MustParse("0.23.0"),

			wantErr: true,
		},
		{
			name: "testing kubectl",
			args: args{
				ctx:  context.Background(),
				path: "kustomize",
			},
			want:    semver.MustParse("5.3.0"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionFromBinary(tt.args.ctx, tt.args.path)
			if (err != nil) && tt.wantErr {
				t.Errorf("ParseVersionFromBinary() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseVersionFromBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVersionFromImage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		runtime   string
		image     string
		command   string
		expected  string
		shouldErr bool
	}{
		{
			name:      "withoutErr",
			runtime:   "docker",
			image:     "myrepo/myimage:1.2.3",
			command:   "",
			expected:  "1.2.3",
			shouldErr: false,
		},
		{
			name:      "withErr",
			runtime:   "docker",
			image:     "myrepo/myimage:invalid",
			command:   "",
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			version, err := ParseVersionFromImage(ctx, tt.runtime, tt.image, tt.command)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if version.String() != tt.expected {
				t.Errorf("expected version %s but got %s", tt.expected, version.String())
			}
		})
	}
}
