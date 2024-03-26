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

// Package components contains a command to list existing components.
package components

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/printers"
)

type flagpole struct {
	Name   string
	Output string
}

// NewCommand returns a new cobra.Command for get components
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "components",
		Short: "List components",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVarP(&flags.Output, "output", "o", "name", "Output format (name, wide)")
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
			logger.Warn("Cluster is not exists")
		}
		return err
	}

	components, err := rt.ListComponents(ctx)
	if err != nil {
		return err
	}

	switch flags.Output {
	default:
		return fmt.Errorf("unknown output format %q", flags.Output)
	case "name":
		for _, component := range components {
			fmt.Println(component.Name)
		}
	case "wide":
		records := [][]string{
			{"NAME", "STATUS"},
		}

		for _, component := range components {
			s, err := rt.InspectComponent(ctx, component.Name)
			if err != nil {
				records = append(records, []string{component.Name, "Error:" + err.Error()})
				continue
			}
			switch s {
			default:
				records = append(records, []string{component.Name, "Unknown"})
			case runtime.ComponentStatusReady:
				records = append(records, []string{component.Name, "Ready"})
			case runtime.ComponentStatusRunning:
				records = append(records, []string{component.Name, "NotReady"})
			case runtime.ComponentStatusStopped:
				records = append(records, []string{component.Name, "Stopped"})
			}
		}

		w := printers.NewTablePrinter(os.Stdout)
		err := w.WriteAll(records)
		if err != nil {
			return err
		}
	}
	return nil
}
