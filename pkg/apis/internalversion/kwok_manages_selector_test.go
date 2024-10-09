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

package internalversion

import (
	"reflect"
	"testing"
)

func Test_parseManagesSelector(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    *ManagesSelector
		wantErr bool
	}{
		{
			name:    "empty",
			args:    "",
			wantErr: true,
		},
		{
			name: "pod",
			args: "pod",
			want: &ManagesSelector{
				Kind: "pod",
			},
		},
		{
			name: "pod v1",
			args: "pod/v1",
			want: &ManagesSelector{
				Kind:    "pod",
				Version: "v1",
			},
		},
		{
			name: "deploy.apps",
			args: "deploy.apps",
			want: &ManagesSelector{
				Kind:  "deploy",
				Group: "apps",
			},
		},
		{
			name: "deploy.apps v1",
			args: "deploy.apps/v1",
			want: &ManagesSelector{
				Kind:    "deploy",
				Group:   "apps",
				Version: "v1",
			},
		},
		{
			name: "pod name=po",
			args: "pod:metadata.name=po",
			want: &ManagesSelector{
				Kind: "pod",
				Name: "po",
			},
		},
		{
			name: "pod labels.apps.group=xxx",
			args: "pod:metadata.labels.apps.group=xxx",
			want: &ManagesSelector{
				Kind: "pod",
				Labels: map[string]string{
					"apps.group": "xxx",
				},
			},
		},
		{
			name: "pod annotations.apps.group=xxx",
			args: "pod:metadata.annotations.apps.group=xxx",
			want: &ManagesSelector{
				Kind: "pod",
				Annotations: map[string]string{
					"apps.group": "xxx",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseManagesSelector(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTargetResourceRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTargetResourceRef() got = %v, want %v", got, tt.want)
			}

			rev := got.String()
			if rev != tt.args {
				t.Errorf("reverse got = %v, want %v", rev, tt.args)
			}
		})
	}
}
