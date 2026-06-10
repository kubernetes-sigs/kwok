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

// Package etcdctl provides a method for users to use the etcdctl binary in a cluster.
package etcdctl

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilscompletion "sigs.k8s.io/kwok/pkg/utils/completion"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for use etcdctl in a cluster
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Use:                "etcdctl [command]",
		Short:              "Run etcdctl in cluster",
		GroupID:            "tool",
		DisableFlagParsing: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			flags.Name = config.DefaultCluster
			name := flags.Name
			workdir := utilspath.Join(config.ClustersDir, flags.Name)

			logger := log.FromContext(ctx)
			logger = logger.With(
				"cluster", flags.Name,
			)
			ctx = log.NewContext(ctx, logger)

			rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completeArgs = make([]string, 0, len(args)+2)
			completeArgs = append(completeArgs, "__complete")
			completeArgs = append(completeArgs, args...)
			completeArgs = append(completeArgs, toComplete)

			var buf bytes.Buffer
			_ = rt.EtcdctlInCluster(utilsexec.WithWriteTo(ctx, &buf), completeArgs...)

			return utilscompletion.ParseCobraOutput(buf.String())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := flags.Name
	workdir := utilspath.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With(
		"cluster", flags.Name,
	)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		return err
	}

	err = rt.EtcdctlInCluster(utilsexec.WithStdIO(ctx), args...)

	if err != nil {
		return err
	}
	return nil
}
