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

// Package cluster contains a command to delete a cluster.
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
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name       string
	Kubeconfig string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.Kubeconfig = path.RelFromHome(kubeconfig.GetRecommendedKubeconfigPath())

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "The path to the kubeconfig file that will remove the deleted cluster")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	var err error
	flags.Kubeconfig, err = path.Expand(flags.Kubeconfig)
	if err != nil {
		return err
	}

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	// Stop the cluster
	start := time.Now()
	logger.Info("Cluster is stopping")
	err = rt.Down(ctx)
	if err != nil {
		return err
	}
	logger.Info("Cluster is stopped",
		"elapsed", time.Since(start),
	)

	// Delete the cluster
	start = time.Now()
	logger.Info("Cluster is deleting")
	if flags.Kubeconfig != "" {
		err = rt.RemoveContext(ctx, flags.Kubeconfig)
		if err != nil {
			logger.Error("Failed to remove context from kubeconfig", err,
				"kubeconfig", flags.Kubeconfig,
			)
		}
		logger.Debug("Remove context from kubeconfig",
			"kubeconfig", flags.Kubeconfig,
		)
	}
	err = rt.Uninstall(ctx)
	if err != nil {
		return err
	}
	logger.Info("Cluster is deleted",
		"elapsed", time.Since(start),
	)
	return nil
}
