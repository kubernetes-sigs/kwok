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

package artifacts

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name   string
	Filter string

	internalversion.KwokctlConfigurationOptions
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.KwokctlConfigurationOptions = config.GetKwokctlConfiguration(ctx).Options

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "artifacts",
		Short: "Lists binaries or images used by cluster",
		Long:  "Lists binaries or images used by cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.Runtime, "runtime", flags.Runtime, fmt.Sprintf("Runtime of the cluster (%s)", strings.Join(runtime.DefaultRegistry.List(), " or ")))
	cmd.Flags().StringVar(&flags.Filter, "filter", flags.Filter, "Filter the list of (binary or image)")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	buildRuntime, ok := runtime.DefaultRegistry.Get(flags.Runtime)
	if !ok {
		return fmt.Errorf("runtime %q not found", flags.Runtime)
	}

	rt, err := buildRuntime(name, workdir)
	if err != nil {
		return err
	}
	artifacts := []string{}

	_, err = rt.Config(ctx)
	if err != nil {
		err = rt.SetConfig(ctx, &flags.KwokctlConfigurationOptions)
		if err != nil {
			return err
		}
	}
	if flags.Filter == "" || flags.Filter == "binary" {
		binaries, err := rt.ListBinaries(ctx)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, binaries...)
	}
	if flags.Filter == "" || flags.Filter == "image" {
		images, err := rt.ListImages(ctx)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, images...)
	}

	sort.Strings(artifacts)

	if len(artifacts) == 0 {
		if flags.Filter == "" {
			logger.Info("No artifacts found",
				"runtime", flags.Runtime,
			)
		} else {
			logger.Info("No artifacts found",
				"runtime", flags.Runtime,
				"filter", flags.Filter,
			)
		}
	} else {
		for _, artifact := range artifacts {
			fmt.Println(artifact)
		}
	}
	return nil
}
