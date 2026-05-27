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

package completion

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestNoFileCompletions(t *testing.T) {
	completions, directive := NoFileCompletions(nil, nil, "")
	if completions != nil {
		t.Errorf("NoFileCompletions() completions = %v, want nil", completions)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("NoFileCompletions() directive = %v, want %v", directive, cobra.ShellCompDirectiveNoFileComp)
	}
}

func TestFixedCompletions(t *testing.T) {
	tests := []struct {
		name      string
		choices   []string
		want      []string
		directive cobra.ShellCompDirective
	}{
		{
			name:      "nil choices",
			choices:   nil,
			want:      nil,
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:      "single choice",
			choices:   []string{"foo"},
			want:      []string{"foo"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:      "multiple choices",
			choices:   []string{"foo", "bar", "baz"},
			want:      []string{"foo", "bar", "baz"},
			directive: cobra.ShellCompDirectiveNoFileComp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := FixedCompletions(tt.choices)
			gotCompletions, gotDirective := fn(nil, nil, "")
			if !reflect.DeepEqual(gotCompletions, tt.want) {
				t.Errorf("FixedCompletions()() completions = %v, want %v", gotCompletions, tt.want)
			}
			if gotDirective != tt.directive {
				t.Errorf("FixedCompletions()() directive = %v, want %v", gotDirective, tt.directive)
			}
		})
	}
}

func TestParseCobraOutput(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		wantCompletions []string
		wantDirective   cobra.ShellCompDirective
	}{
		{
			name:            "empty output",
			output:          "",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:            "only directive",
			output:          ":4\n",
			wantCompletions: nil,
			wantDirective:   cobra.ShellCompDirective(4),
		},
		{
			name:            "completions with directive",
			output:          "foo\nbar\n:0\n",
			wantCompletions: []string{"foo", "bar"},
			wantDirective:   cobra.ShellCompDirective(0),
		},
		{
			name:            "completions with descriptions",
			output:          "foo\tsome description\nbar\tbaz\n:4\n",
			wantCompletions: []string{"foo\tsome description", "bar\tbaz"},
			wantDirective:   cobra.ShellCompDirective(4),
		},
		{
			name:            "completions without directive",
			output:          "foo\nbar\n",
			wantCompletions: []string{"foo", "bar"},
			wantDirective:   cobra.ShellCompDirectiveNoFileComp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := ParseCobraOutput(tt.output)
			if !reflect.DeepEqual(completions, tt.wantCompletions) {
				t.Errorf("ParseCobraOutput() completions = %v, want %v", completions, tt.wantCompletions)
			}
			if directive != tt.wantDirective {
				t.Errorf("ParseCobraOutput() directive = %v, want %v", directive, tt.wantDirective)
			}
		})
	}
}
