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

package kubectl

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/logger"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logger.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Use:          "kubectl",
		Short:        "kubectl in cluster",
		Long:         "kubectl in cluster",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			err := runE(cmd.Context(), logger, flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}
	cmd.DisableFlagParsing = true
	return cmd
}

func runE(ctx context.Context, logger logger.Logger, flags *flagpole, args []string) error {
	name := fmt.Sprintf("%s-%s", vars.ProjectName, flags.Name)
	workdir := utils.PathJoin(vars.ClustersDir, flags.Name)

	rt, err := runtime.DefaultRegistry.Load(name, workdir, logger)
	if err != nil {
		return err
	}

	err = rt.KubectlInCluster(ctx, utils.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}, args...)

	if err != nil {
		return err
	}
	return nil
}
