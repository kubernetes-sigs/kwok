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

package clusters

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/logger"
)

// NewCommand returns a new cobra.Command for getting the list of clusters
func NewCommand(logger logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "clusters",
		Short: "Lists existing clusters by their name",
		Long:  "Lists existing clusters by their name",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(logger)
		},
	}
	return cmd
}

func runE(logger logger.Logger) error {
	clusters, err := runtime.ListClusters(vars.ClustersDir)
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		logger.Printf("no clusters found")
	} else {
		for _, cluster := range clusters {
			fmt.Println(cluster)
		}
	}
	return nil
}
