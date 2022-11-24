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

package save

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/log"
)

type flagpole struct {
	Name   string
	Path   string
	Format string
}

// NewCommand returns a new cobra.Command for cluster snapshotting.
func NewCommand(logger *log.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "save",
		Short: "Save the snapshot of the cluster",
		Long:  "Save the snapshot of the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the snapshot")
	cmd.Flags().StringVar(&flags.Format, "format", "etcd", "Format of the snapshot file (etcd)")
	return cmd
}

func runE(ctx context.Context, logger *log.Logger, flags *flagpole) error {
	name := fmt.Sprintf("%s-%s", vars.ProjectName, flags.Name)
	workdir := utils.PathJoin(vars.ClustersDir, flags.Name)

	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}

	rt, err := runtime.DefaultRegistry.Load(name, workdir, logger)
	if err != nil {
		return err
	}

	if flags.Format == "etcd" {
		err = rt.SnapshotSave(ctx, flags.Path)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupport format %q", flags.Format)
	}
	return nil
}
