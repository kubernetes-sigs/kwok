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
	"fmt"
	"testing"
)

func TestTrimPrefixV(t *testing.T) {
	testCases := []struct {
		version  string
		expected string
	}{
		{
			version:  "v1.0.0",
			expected: "1.0.0",
		},
		{
			version:  "v0.1.0",
			expected: "0.1.0",
		},
		{
			version:  "0.1.0",
			expected: "0.1.0",
		},
		{
			version:  "v1.0",
			expected: "1.0",
		},
		{
			version:  "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TrimPrefixV(%s)", tc.version), func(t *testing.T) {
			result := TrimPrefixV(tc.version)
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestAddPrefixV(t *testing.T) {
	testCases := []struct {
		version  string
		expected string
	}{
		{
			version:  "1.0.0",
			expected: "v1.0.0",
		},
		{
			version:  "0.1.0",
			expected: "v0.1.0",
		},
		{
			version:  "v0.1.0",
			expected: "v0.1.0",
		},
		{
			version:  "v1.0",
			expected: "v1.0",
		},
		{
			version:  "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("AddPrefixV(%s)", tc.version), func(t *testing.T) {
			result := AddPrefixV(tc.version)
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}
