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

// Package restore provides a command to restore the snapshot of a cluster.
package restore

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name    string
	Path    string
	Format  string
	Filters []string
}

// NewCommand returns a new cobra.Command to restore the cluster as a snapshot.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "restore",
		Short: "Restore the snapshot of the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the snapshot")
	cmd.Flags().StringVar(&flags.Format, "format", "etcd", "Format of the snapshot file (etcd, k8s)")
	cmd.Flags().StringSliceVar(&flags.Filters, "filter", snapshot.Resources, "Filter the resources to restore, only support for k8s format")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)
	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}
	if !file.Exists(flags.Path) {
		return fmt.Errorf("path %q does not exist", flags.Path)
	}

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster is not exists")
		}
		return err
	}

	switch flags.Format {
	case "etcd":
		err = rt.SnapshotRestore(ctx, flags.Path)
		if err != nil {
			return err
		}
	case "k8s":
		err = rt.SnapshotRestoreWithYAML(ctx, flags.Path, runtime.SnapshotRestoreWithYAMLConfig{
			Filters: flags.Filters,
		})
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupport format %q", flags.Format)
	}
	return nil
}
