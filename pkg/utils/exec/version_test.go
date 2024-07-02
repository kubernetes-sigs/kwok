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

package exec

import (
	"context"
	"testing"
)

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
