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
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		return err
	}
	logger.Info("Stopping cluster")
	err = rt.Down(ctx)
	if err != nil {
		logger.Error("Stopping cluster", err)
	}

	logger.Info("Deleting cluster")
	err = rt.Uninstall(ctx)
	if err != nil {
		return err
	}
	logger.Info("Cluster deleted")

	// load kubeconfig file
	kubeconfig, err := clientcmd.LoadFromFile(os.Getenv("KUBECONFIG"))
	if err != nil {
		fmt.Errorf("failed to load kubeconfig file: %w", err)
	}
	// switch back to the previous context using a defer statement
	defer func() {
		kubeconfig.CurrentContext = config.OriginCluster
		if err := clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *kubeconfig, false); err != nil {
			fmt.Println("failed to switch back to the previous context")
		}
		fmt.Println("Switched back to the previous context")
	}()

	return nil
}
