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

package logs

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/logger"
)

type flagpole struct {
	Name   string
	Follow bool
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logger.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.ExactArgs(1),
		Use:   "logs",
		Short: "Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]",
		Long:  "Logs one of [audit, etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kwok-controller, prometheus]",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags, args)
		},
	}
	cmd.Flags().BoolVarP(&flags.Follow, "follow", "f", false, "Specify if the logs should be streamed")
	return cmd
}

func runE(ctx context.Context, logger logger.Logger, flags *flagpole, args []string) error {
	name := fmt.Sprintf("%s-%s", vars.ProjectName, flags.Name)
	workdir := utils.PathJoin(vars.ClustersDir, flags.Name)

	rt, err := runtime.DefaultRegistry.Load(name, workdir, logger)
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
