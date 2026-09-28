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

package port_forward

import "testing"

func TestParseLocalPort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint32
		wantErr bool
	}{
		{name: "ordinary port", input: "8080", want: 8080},
		{name: "leading zero is not octal", input: "010", want: 10},
		{name: "leading zero remains decimal", input: "08080", want: 8080},
		{name: "maximum port", input: "65535", want: 65535},
		{name: "port above maximum", input: "65536", wantErr: true},
		{name: "port wraps uint32", input: "4294967296", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLocalPort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLocalPort(%q) error = %v, wantErr %t", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseLocalPort(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
