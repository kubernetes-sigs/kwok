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

package cmd

import (
	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/create"
	del "sigs.k8s.io/kwok/pkg/kwokctl/cmd/delete"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/get"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/kubectl"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/logs"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/log"
)

// NewCommand returns a new cobra.Command for root
func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:          cobra.NoArgs,
		Use:           "kwokctl [command]",
		Short:         "Kwokctl is a Kwok cluster management tool",
		Long:          "Kwokctl is a Kwok cluster management tool",
		Version:       consts.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&vars.DefaultCluster, "name", vars.DefaultCluster, "cluster name")
	cmd.TraverseChildren = true

	cmd.AddCommand(
		create.NewCommand(logger),
		del.NewCommand(logger),
		get.NewCommand(logger),
		kubectl.NewCommand(logger),
		logs.NewCommand(logger),
		snapshot.NewCommand(logger),
	)
	return cmd
}
