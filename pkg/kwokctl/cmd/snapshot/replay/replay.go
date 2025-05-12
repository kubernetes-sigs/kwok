/*
Copyright 2024 The Kubernetes Authors.

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

// Package replay provides a command to replay the recordingof a cluster.
package replay

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type flagpole struct {
	Name     string
	Path     string
	Snapshot bool
}

// NewCommand returns a new cobra.Command to replay the cluster as a recording.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "replay",
		Short: "Replay the recording to the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the recording")
	cmd.Flags().BoolVar(&flags.Snapshot, "snapshot", false, "Only restore the snapshot")
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
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	components, err := rt.ListComponents(ctx)
	if err != nil {
		return err
	}

	components = slices.Filter(components, func(component internalversion.Component) bool {
		return component.Name != consts.ComponentKubeApiserver && component.Name != consts.ComponentEtcd
	})

	for _, component := range components {
		err = rt.StopComponent(ctx, component.Name)
		if err != nil {
			logger.Error("Failed to stop component", err,
				"component", component.Name,
			)
		}
	}

	defer func() {
		for _, component := range components {
			err = rt.StartComponent(ctx, component.Name)
			if err != nil {
				logger.Error("Failed to start component", err,
					"component", component.Name,
				)
			}
		}
	}()

	err = rt.KectlInCluster(exec.WithStdIO(ctx),
		"snapshot",
		"replay",
		"--path="+flags.Path,
		"--snapshot="+format.String(flags.Snapshot),
	)

	if err != nil {
		return err
	}
	return nil
}
