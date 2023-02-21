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

package path

import (
	"os"
	"testing"
)

func TestExpand(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	var testCases = []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "tilde and dot",
			input:    "~/./example.txt",
			expected: Join(home, "example.txt"),
			wantErr:  false,
		},
		{
			name:     "tilde and dotdot",
			input:    "~/../example.txt",
			expected: Join(home, "../example.txt"),
			wantErr:  false,
		},
		{
			name:     "tilde",
			input:    "~",
			expected: home,
			wantErr:  false,
		},
		{
			name:     "tilde slash",
			input:    "~/example.txt",
			expected: Join(home, "example.txt"),
			wantErr:  false,
		},
		{
			name:     "absolute path",
			input:    "/example.txt",
			expected: "/example.txt",
			wantErr:  false,
		},
		{
			name:     "pwd path",
			input:    "example.txt",
			expected: Join(wd, "example.txt"),
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := Expand(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if output != tc.expected {
				t.Errorf("Expected %s, but got %s", tc.expected, output)
			}
		})
	}
}
