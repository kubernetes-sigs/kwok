/*
Copyright 2023 The Kubernetes Authors.

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

// Package export is the export of external cluster
package export

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
)

type flagpole struct {
	Path              string
	Kubeconfig        string
	Filters           []string
	ImpersonateUser   string
	ImpersonateGroups []string
	PageSize          int64
	PageBufferSize    int32
}

// NewCommand returns a new cobra.Command for cluster exporting.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "export",
		Short: "[experimental] Export the snapshots of external clusters",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags)
		},
	}
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "Path to the kubeconfig file to use")
	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the snapshot")
	cmd.Flags().StringSliceVar(&flags.Filters, "filter", snapshot.Resources, "Filter the resources to export")
	cmd.Flags().StringVar(&flags.ImpersonateUser, "as", "", "Username to impersonate for the operation. User could be a regular user or a service account in a namespace.")
	cmd.Flags().StringSliceVar(&flags.ImpersonateGroups, "as-group", nil, "Group to impersonate for the operation, this flag can be repeated to specify multiple groups.")
	cmd.Flags().Int64Var(&flags.PageSize, "page-size", 500, "Define the page size")
	cmd.Flags().Int32Var(&flags.PageBufferSize, "page-buffer-size", 10, "Define the number of pages to buffer")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}
	if _, err := os.Stat(flags.Path); err == nil {
		return fmt.Errorf("file %q already exists", flags.Path)
	}

	file, err := os.Create(flags.Path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(flags.Path)
		}
	}()

	impersonateConfig := rest.ImpersonationConfig{
		UserName: flags.ImpersonateUser,
		Groups:   flags.ImpersonateGroups,
	}

	pagerConfig := &snapshot.PagerConfig{
		PageSize:       flags.PageSize,
		PageBufferSize: flags.PageBufferSize,
	}

	snapshotSaveConfig := snapshot.SaveConfig{
		PagerConfig:         pagerConfig,
		ImpersonationConfig: impersonateConfig,
	}

	err = snapshot.Save(ctx, flags.Kubeconfig, file, flags.Filters, snapshotSaveConfig)
	if err != nil {
		return err
	}

	return nil
}
