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

// Package snapshot contains a parent command which snapshots one of cluster.
package snapshot

import (
	"context"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot/export"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot/record"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot/replay"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot/restore"
	"sigs.k8s.io/kwok/pkg/kwokctl/cmd/snapshot/save"
)

// NewCommand returns a new cobra.Command for cluster snapshot
func NewCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "snapshot [command]",
		Short: "[experimental] Snapshot [save, restore, record, replay, export] one of cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(save.NewCommand(ctx))
	cmd.AddCommand(restore.NewCommand(ctx))
	cmd.AddCommand(export.NewCommand(ctx))
	cmd.AddCommand(replay.NewCommand(ctx))
	cmd.AddCommand(record.NewCommand(ctx))
	return cmd
}
