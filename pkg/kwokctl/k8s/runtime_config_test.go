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

package k8s

import (
	"testing"
)

func TestGetRuntimeConfig(t *testing.T) {
	tests := []struct {
		name     string
		version  int
		expected string
	}{
		{"Version less than 17", 16, ""},
		{"Version equal to 17", 17, "api/legacy=false,api/alpha=false"},
		{"Version greater than 17", 18, "api/legacy=false,api/alpha=false"},
		{"Negative version", -1, ""},
		{"Zero version", 0, ""},
		{"High version number", 100, "api/legacy=false,api/alpha=false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRuntimeConfig(tt.version)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
