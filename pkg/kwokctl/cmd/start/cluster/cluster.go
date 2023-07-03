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

// Package cluster implements the start cluster command
package cluster

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name    string
	Wait    time.Duration
	Timeout time.Duration
}

// NewCommand returns a new cobra.Command for start cluster
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Start a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 0, "Timeout for waiting for the cluster to be started")
	cmd.Flags().DurationVar(&flags.Wait, "wait", 0, "Wait for the cluster to be ready")

	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	gctx := ctx
	if flags.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, flags.Timeout)
		defer cancel()
	}

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster is not exists")
		}
		return err
	}

	start := time.Now()
	logger.Info("Cluster is starting")
	err = rt.Start(ctx)
	if err != nil {
		return err
	}
	logger.Info("Cluster is started",
		"elapsed", time.Since(start),
	)

	if flags.Wait > 0 {
		start := time.Now()
		logger.Info("Waiting for cluster to be ready")
		err = rt.WaitReady(gctx, flags.Wait)
		if err != nil {
			logger.Error("Failed to wait for cluster to be ready", err,
				"elapsed", time.Since(start),
			)
		} else {
			logger.Info("Cluster is ready",
				"elapsed", time.Since(start),
			)
		}
	}

	return nil
}
