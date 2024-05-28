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

func TestGetEtcdVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  int
		expected string
	}{
		{"Version 8", 8, "3.0.17"},
		{"Version 9", 9, "3.1.12"},
		{"Version 10", 10, "3.1.12"},
		{"Version 11", 11, "3.2.18"},
		{"Version 12", 12, "3.2.24"},
		{"Version 13", 13, "3.2.24"},
		{"Version 14", 14, "3.3.10"},
		{"Version 15", 15, "3.3.10"},
		{"Version 16", 16, "3.3.17-0"},
		{"Version 17", 17, "3.4.3-0"},
		{"Version 18", 18, "3.4.3-0"},
		{"Version 19", 19, "3.4.13-0"},
		{"Version 20", 20, "3.4.13-0"},
		{"Version 21", 21, "3.4.13-0"},
		{"Version 22", 22, "3.5.11-0"},
		{"Version 23", 23, "3.5.11-0"},
		{"Version 24", 24, "3.5.11-0"},
		{"Version 25", 25, "3.5.11-0"},
		{"Version 26", 26, "3.5.11-0"},
		{"Version 27", 27, "3.5.11-0"},
		{"Version 28", 28, "3.5.11-0"},
		{"Version 29", 29, "3.5.11-0"},
		{"Version too low", 7, "3.0.17"},
		{"Version too high", 30, "3.5.11-0"},
		{"Negative version", -1, "3.5.11-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEtcdVersion(tt.version)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
