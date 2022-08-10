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

package cluster

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/logger"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logger.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Deletes a cluster",
		Long:  "Deletes a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags)
		},
	}
	return cmd
}

func runE(ctx context.Context, logger logger.Logger, flags *flagpole) error {
	name := fmt.Sprintf("%s-%s", vars.ProjectName, flags.Name)
	workdir := utils.PathJoin(vars.ClustersDir, flags.Name)

	rt, err := runtime.DefaultRegistry.Load(name, workdir, logger)
	if err != nil {
		return err
	}
	logger.Printf("Stopping cluster %q", name)
	err = rt.Down(ctx)
	if err != nil {
		logger.Printf("Error stopping cluster %q: %v", name, err)
	}

	logger.Printf("Deleting cluster %q", name)
	err = rt.Uninstall(ctx)
	if err != nil {
		return err
	}
	logger.Printf("Cluster %q deleted", name)
	return nil
}
