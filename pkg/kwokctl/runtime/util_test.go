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

package runtime

import (
	"reflect"
	"testing"
)

func Test_sortArgsOnCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty",
			args: []string{},
			want: []string{},
		},
		{
			name: "sub command",
			args: []string{"sub"},
			want: []string{"sub"},
		},
		{
			name: "a param",
			args: []string{"--foo"},
			want: []string{"--foo"},
		},
		{
			name: "params unsorted",
			args: []string{"--foo", "--bar"},
			want: []string{"--bar", "--foo"},
		},
		{
			name: "params sorted",
			args: []string{"--bar", "--foo"},
			want: []string{"--bar", "--foo"},
		},
		{
			name: "subcommand between in params",
			args: []string{"--foo", "--bar", "sub", "--foo", "--bar"},
			want: []string{"--bar", "--foo", "sub", "--bar", "--foo"},
		},
		{
			name: "-- between in params",
			args: []string{"--foo", "--bar", "--", "--foo", "--bar"},
			want: []string{"--bar", "--foo", "--", "--bar", "--foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortArgsOnCommand(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortArgsOnCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
