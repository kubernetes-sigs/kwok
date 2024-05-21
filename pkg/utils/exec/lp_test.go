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

package exec

import (
	"testing"
)

func TestLookPath(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		expectErr bool
	}{
		{"ExistingExecutable", "ls", false},                   // A common Unix command
		{"NonExistingExecutable", "nonexistentcommand", true}, // A command that does not exist
		{"EmptyString", "", true},                             // An empty string
		{"PathToExecutable", "/usr/bin/cat", false},           // Full path to an existing executable
		{"PathToNonExecutableFile", "/etc/hosts", true},       // Full path to a non-executable file
		{"Directory", "/usr/bin", true},                       // A directory
		{"SpecialCharacters", "weird$cmd&name", true},         // Command na              // A relative path to an existing executable
		{"PathToNonExecutableDirectory", "/usr/lib", true},    // Full path to a non-executable directory
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LookPath(tt.file)
			if (err != nil) != tt.expectErr {
				t.Fatalf("LookPath(%s) error = %v, expectErr %v", tt.file, err, tt.expectErr)
			}
		})
	}
}
