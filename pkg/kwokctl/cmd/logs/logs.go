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
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name   string
	Follow bool
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]",
		Long:  "Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags, args)
		},
	}
	cmd.Flags().BoolVarP(&flags.Follow, "follow", "f", false, "Specify if the logs should be streamed")
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
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
