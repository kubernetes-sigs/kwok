/*
Copyright 2026 The Kubernetes Authors.

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

package kind

import (
	"reflect"
	"strings"
	"testing"
)

func Test_buildKindConfigV1alpha4_FeatureGates(t *testing.T) {
	tests := []struct {
		name         string
		featureGates []string
		want         map[string]bool
		wantErr      bool
	}{
		{
			name:         "normal valid values",
			featureGates: []string{"Foo=true", "Bar=false"},
			want:         map[string]bool{"Foo": true, "Bar": false},
		},
		{
			name:         "valid alternate strconv.ParseBool values",
			featureGates: []string{"Foo=T", "Bar=0"},
			want:         map[string]bool{"Foo": true, "Bar": false},
		},
		{
			name:         "missing equals",
			featureGates: []string{"Foo"},
			wantErr:      true,
		},
		{
			name:         "empty value",
			featureGates: []string{"Foo="},
			wantErr:      true,
		},
		{
			name:         "invalid value",
			featureGates: []string{"Foo=ture"},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildKindConfigV1alpha4(BuildKindConfig{FeatureGates: tt.featureGates})
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildKindConfigV1alpha4() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got.FeatureGates, tt.want) {
				t.Errorf("buildKindConfigV1alpha4() FeatureGates = %v, want %v", got.FeatureGates, tt.want)
			}
		})
	}
}

func Test_buildKindConfigV1alpha4_FeatureGates_InvalidValueError(t *testing.T) {
	_, err := buildKindConfigV1alpha4(BuildKindConfig{FeatureGates: []string{"Foo=ture"}})
	if err == nil {
		t.Fatal("buildKindConfigV1alpha4() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid feature gate") {
		t.Errorf("buildKindConfigV1alpha4() error = %q, want it to contain %q", err.Error(), "invalid feature gate")
	}
	if !strings.Contains(err.Error(), "Foo=ture") {
		t.Errorf("buildKindConfigV1alpha4() error = %q, want it to contain %q", err.Error(), "Foo=ture")
	}
}
