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

// Package config provides the kwokctl config command.
package config

import (
	"context"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/config/convert"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/config/reset"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/config/tidy"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/config/view"
)

// NewCommand returns a new cobra.Command for config
func NewCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "config [command]",
		Short: "Manage [reset, tidy, view] default config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(reset.NewCommand(ctx))
	cmd.AddCommand(tidy.NewCommand(ctx))
	cmd.AddCommand(view.NewCommand(ctx))
	cmd.AddCommand(convert.NewCommand(ctx))
	return cmd
}
