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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "clusters",
		Short: "Lists existing clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context())
		},
	}
	return cmd
}

func runE(ctx context.Context) error {
	clusters, err := runtime.ListClusters(config.ClustersDir)
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		logger := log.FromContext(ctx)
		logger.Info("No clusters found")
	} else {
		for _, cluster := range clusters {
			_, _ = fmt.Println(cluster)
		}
	}
	return nil
}
