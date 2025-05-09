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

// Package get defines a command to get data in etcd
package get

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name      string
	Namespace string
	Output    string
	ChunkSize int64
	Watch     bool
	WatchOnly bool
}

// NewCommand returns a new cobra.Command for use hack the etcd data.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "get [resource] [name]",
		Short: "get data in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "yaml", "output format. One of: (json, yaml, raw, key).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().BoolVarP(&flags.Watch, "watch", "w", false, "after listing/getting the requested object, watch for changes")
	cmd.Flags().BoolVar(&flags.WatchOnly, "watch-only", false, "watch for changes to the requested object(s), without listing/getting first")
	cmd.Flags().Int64Var(&flags.ChunkSize, "chunk-size", 500, "chunk size of the list pager")
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
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	err = rt.KectlInCluster(exec.WithStdIO(ctx),
		append([]string{"get",
			"--output=" + flags.Output,
			"--namespace=" + flags.Namespace,
			"--watch=" + strconv.FormatBool(flags.Watch),
			"--watch-only=" + strconv.FormatBool(flags.WatchOnly),
			"--chunk-size=" + strconv.FormatInt(flags.ChunkSize, 10),
		}, args...)...,
	)

	if err != nil {
		return err
	}
	return nil
}
