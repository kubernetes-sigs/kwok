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
	All        bool
	Force      bool
}

// NewCommand returns a new cobra.Command for cluster deletion
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.Kubeconfig = path.RelFromHome(kubeconfig.GetRecommendedKubeconfigPath())
	flags.All = false
	flags.Force = false

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flags.All {
				flags.Name = config.DefaultCluster
			}
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "The path to the kubeconfig file that will remove the deleted cluster")
	cmd.Flags().BoolVar(&flags.All, "all", flags.All, "Delete all clusters managed by kwokctl")
	cmd.Flags().BoolVar(&flags.Force, "force", flags.Force, "Delete cluster depending on runtime availability")

	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	var clusters []string
	var err error

	if flags.All {
		clusters, err = runtime.ListClusters(ctx)
		if err != nil {
			return err
		}
		for _, cluster := range clusters {
			err = deleteCluster(ctx, cluster, flags.Kubeconfig, flags)
			if err != nil {
				return err
			}
		}
	} else {
		err = deleteCluster(ctx, flags.Name, flags.Kubeconfig, flags)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteCluster(ctx context.Context, clusterName string, kubeconfigPath string, flags *flagpole) error {
	name := config.ClusterName(clusterName)
	workdir := path.Join(config.ClustersDir, clusterName)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", clusterName)
	ctx = log.NewContext(ctx, logger)

	var err error
	kubeconfigPath, err = path.Expand(kubeconfigPath)
	if err != nil {
		return err
	}

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
			return nil
		}
		return err
	}

	err = rt.Available(ctx)
	if err != nil {
		if !flags.Force {
			var err error
			return err
		}
		logger.Warn("Unavailable runtime but proceed with force delete")
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
	if kubeconfigPath != "" {
		err = rt.RemoveContext(ctx, kubeconfigPath)
		if err != nil {
			logger.Error("Failed to remove context from kubeconfig", err,
				"kubeconfig", kubeconfigPath,
			)
		}
		logger.Debug("Remove context from kubeconfig",
			"kubeconfig", kubeconfigPath,
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
