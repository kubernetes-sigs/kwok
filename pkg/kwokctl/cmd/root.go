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

// Package cmd defines a root command for the kwokctl.
package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	conf "sigs.k8s.io/kwok/pkg/kwokctl/cmd/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/create"
	del "sigs.k8s.io/kwok/pkg/kwokctl/cmd/delete"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/etcdctl"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/export"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/get"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/hack"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/kubectl"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/logs"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/portforward"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/scale"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/start"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/stop"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// NewCommand returns a new cobra.Command for root
func NewCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Args:          cobra.NoArgs,
		Use:           "kwokctl [command]",
		Short:         "kwokctl is a tool to streamline the creation and management of clusters, with nodes simulated by kwok",
		Version:       version.DisplayVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&config.DefaultCluster, "name", config.DefaultCluster, "cluster name")
	cmd.PersistentFlags().BoolVar(&dryrun.DryRun, "dry-run", dryrun.DryRun, "Print the command that would be executed, but do not execute it")
	cmd.TraverseChildren = true

	cmd.AddCommand(
		conf.NewCommand(ctx),
		create.NewCommand(ctx),
		del.NewCommand(ctx),
		get.NewCommand(ctx),
		start.NewCommand(ctx),
		stop.NewCommand(ctx),
		kubectl.NewCommand(ctx),
		etcdctl.NewCommand(ctx),
		logs.NewCommand(ctx),
		scale.NewCommand(ctx),
		snapshot.NewCommand(ctx),
		export.NewCommand(ctx),
		hack.NewCommand(ctx),
		portforward.NewCommand(ctx),
	)
	return cmd
}
