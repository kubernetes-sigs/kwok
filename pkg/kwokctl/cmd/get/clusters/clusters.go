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

// Package clusters contains a command to list existing clusters.
package clusters

import (
	"context"
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
	Output string
}

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "clusters",
		Short: "Lists existing clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVarP(&flags.Output, "output", "o", "name", "Output format (name, wide)")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	clusters, err := runtime.ListClusters(ctx)
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		if log.IsTerminal() {
			_, _ = fmt.Fprintf(os.Stderr, "No clusters found\n")
		}
	} else {
		switch flags.Output {
		default:
			return fmt.Errorf("unknown output format %q", flags.Output)
		case "name":
			for _, cluster := range clusters {
				_, _ = fmt.Println(cluster)
			}
		case "wide":
			records := [][]string{
				{"NAME", "READY", "STATUS"},
			}

			for _, cluster := range clusters {
				var readyMsg = "0/0"
				var count int
				workdir := path.Join(config.ClustersDir, cluster)
				rt, err := runtime.DefaultRegistry.Load(ctx, cluster, workdir)
				if err != nil {
					records = append(records, []string{cluster, readyMsg, "Failed:" + err.Error()})
					continue
				}

				components, err := rt.ListComponents(ctx)
				if err == nil {
					for _, component := range components {
						s, _ := rt.InspectComponent(ctx, component.Name)
						if s == runtime.ComponentStatusReady {
							count++
						}
					}
					readyMsg = fmt.Sprintf("%d/%d", count, len(components))
				}

				if len(components) == 0 {
					records = append(records, []string{cluster, readyMsg, "Unknown"})
					continue
				}

				if count == 0 {
					records = append(records, []string{cluster, readyMsg, "Stopped"})
					continue
				}

				ready, err := rt.Ready(ctx)
				if err != nil {
					records = append(records, []string{cluster, readyMsg, "Error:" + err.Error()})
					continue
				}

				if !ready {
					records = append(records, []string{cluster, readyMsg, "NotReady"})
					continue
				}

				records = append(records, []string{cluster, readyMsg, "Ready"})
			}

			w := printers.NewTablePrinter(os.Stdout)
			err := w.WriteAll(records)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
