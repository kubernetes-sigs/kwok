/*
Copyright 2025 The Kubernetes Authors.

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

// Package convert provides the kwokctl config convert command.
package convert

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/log"
)

type flagpole struct {
	Write bool
}

// NewCommand returns a new cobra.Command for config convert
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Use:   "convert [file...]",
		Short: "Convert the specified config files to the latest version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags, args)
		},
	}

	cmd.Flags().BoolVarP(&flags.Write, "write", "w", flags.Write, "Write the converted config file back to disk")
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	logger := log.FromContext(ctx)
	for _, f := range args {
		if dryrun.DryRun {
			if flags.Write {
				dryrun.PrintMessage("# Converting and writing back config file %s", f)
			} else {
				dryrun.PrintMessage("# Converting config file %s and outputting to stdout", f)
			}
			continue
		}

		rio, err := config.Load(ctx, f)
		if err != nil {
			logger.Warn("Failed to load config file", "file", f, "error", err)
			continue
		}

		if len(rio) == 0 {
			logger.Warn("No resources found in config file", "file", f)
			continue
		}

		if flags.Write {
			err = config.Save(ctx, f, rio)
			if err != nil {
				return err
			}
		} else {
			err = config.SaveTo(ctx, os.Stdout, rio)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
