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

// Package completion provides helpers for shell completion.
package completion

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// NoFileCompletions can be used to disable file completion for commands that should
// not trigger file completions.
func NoFileCompletions(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// FixedCompletions can be used to create a completion function which always
// returns the same results.
func FixedCompletions(choices []string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return choices, cobra.ShellCompDirectiveNoFileComp
	}
}

// ParseCobraOutput parses the stdout of a cobra __complete invocation into
// completions and a directive. The expected format is one completion per line,
// with the final line being `:N` where N is the ShellCompDirective integer.
func ParseCobraOutput(output string) ([]string, cobra.ShellCompDirective) {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	directive := cobra.ShellCompDirectiveNoFileComp

	if len(lines) == 0 {
		return nil, directive
	}

	// The last line contains the directive in the form ":N" (N is a non-negative integer).
	last := lines[len(lines)-1]
	if strings.HasPrefix(last, ":") {
		if n, err := strconv.Atoi(last[1:]); err == nil && n >= 0 {
			directive = cobra.ShellCompDirective(n)
			lines = lines[:len(lines)-1]
		}
	}

	var completions []string
	for _, line := range lines {
		if line != "" {
			completions = append(completions, line)
		}
	}
	return completions, directive
}
