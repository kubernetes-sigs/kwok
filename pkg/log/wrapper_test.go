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

package log

import (
	"testing"
)

func TestParseLevel(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantL   Level
		wantErr bool
	}{
		{
			name: "debug",
			args: args{
				s: "debug",
			},
			wantL: DebugLevel,
		},
		{
			name: "info",
			args: args{
				s: "info",
			},
			wantL: InfoLevel,
		},
		{
			name: "-4",
			args: args{
				s: "-4",
			},
			wantL: DebugLevel,
		},
		{
			name: "0",
			args: args{
				s: "0",
			},
			wantL: InfoLevel,
		},
		{
			name: "info+1",
			args: args{
				s: "info+1",
			},
			wantL: InfoLevel + 1,
		},
		{
			name: "info-1",
			args: args{
				s: "info-1",
			},
			wantL: InfoLevel - 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotL, err := ParseLevel(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotL != tt.wantL {
				t.Errorf("ParseLevel() gotL = %v, want %v", gotL, tt.wantL)
			}
		})
	}
}
