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

package k8s

import (
	"testing"
)

func TestGetFeatureGates(t *testing.T) {
	// Mocking the rawData and lockEnabled variables for testing
	rawData = []FeatureSpec{
		{Name: "feature1", Stage: GA, Since: 10, Until: 20},
		{Name: "feature2", Stage: Beta, Since: 15, Until: 25},
		{Name: "feature3", Stage: Deprecated, Since: 5, Until: 15},
	}

	lockEnabled = map[string]bool{
		"feature1": true,
		"feature2": false,
		"feature3": true,
	}

	tests := []struct {
		name     string
		version  int
		expected string
	}{{
		name:     "Version 1",
		version:  1,
		expected: "",
	},
		{
			name:     "Version 5",
			version:  5,
			expected: "",
		}, {
			name:     "Version 10",
			version:  10,
			expected: "",
		}, {
			name:     "Version 20",
			version:  20,
			expected: "feature2=false",
		}, {
			name:     "Version 25",
			version:  25,
			expected: "feature2=false",
		}, {
			name:     "Version 30",
			version:  30,
			expected: "",
		}, {
			name:     "Negative Version",
			version:  -1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFeatureGates(tt.version)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
