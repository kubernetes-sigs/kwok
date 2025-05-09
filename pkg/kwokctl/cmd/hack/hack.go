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

// Package hack defines a parent command for hack data
package hack

import (
	"context"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/hack/del"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/hack/get"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/hack/put"
)

// NewCommand returns a new cobra.Command for get
func NewCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Args:       cobra.NoArgs,
		Use:        "hack [command]",
		Short:      "[experimental] Hack [get, put, delete] resources in etcd without apiserver",
		Deprecated: "Use 'kectl' instead for direct etcd interaction.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	// add subcommands
	cmd.AddCommand(get.NewCommand(ctx))
	cmd.AddCommand(del.NewCommand(ctx))
	cmd.AddCommand(put.NewCommand(ctx))
	return cmd
}
