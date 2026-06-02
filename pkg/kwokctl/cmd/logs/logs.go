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

// Package logs contains a command to log a component of a cluster.
package logs

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name   string
	Follow bool
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:    cobra.ExactArgs(1),
		Use:     "logs [component]",
		Short:   "Logs 'audit' (if enabled) or any component name",
		GroupID: "cluster",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			flags.Name = config.DefaultCluster
			name := config.ClusterName(flags.Name)
			workdir := utilspath.Join(config.ClustersDir, flags.Name)

			logger := log.FromContext(ctx)
			logger = logger.With("cluster", flags.Name)
			ctx = log.NewContext(ctx, logger)

			rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			components, err := rt.ListComponents(ctx)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			list := []string{}

			config, err := rt.Config(ctx)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			conf := &config.Options
			if conf.KubeAuditPolicy != "" {
				list = append(list, "audit")
			}

			for _, component := range components {
				list = append(list, component.Name)
			}
			return list, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags, args)
		},
	}
	cmd.Flags().BoolVarP(&flags.Follow, "follow", "f", flags.Follow, "Specify if the logs should be streamed")

	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := utilspath.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	if args[0] == "audit" {
		if flags.Follow {
			err = rt.AuditLogsFollow(ctx, os.Stdout)
		} else {
			err = rt.AuditLogs(ctx, os.Stdout)
		}
	} else {
		if flags.Follow {
			err = rt.LogsFollow(ctx, args[0], os.Stdout)
		} else {
			err = rt.Logs(ctx, args[0], os.Stdout)
		}
	}
	if err != nil {
		return err
	}
	return nil
}
