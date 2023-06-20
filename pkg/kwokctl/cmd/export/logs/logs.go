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

// Package logs implements the `logs` command
package logs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for getting the cluster logs
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.MaximumNArgs(1),
		Use:   "logs [output-dir]",
		Short: "Exports logs to a tempdir or [output-dir] if specified",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(ctx, flags, args)
		},
	}
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx).With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster is not exists")
		}
		return err
	}

	// get the optional directory argument, or create a tempdir under the kwok default working directory
	var dir string
	if len(args) == 0 || args[0] == "" {
		tmp := filepath.Join(workdir, "tmp")
		if err := os.MkdirAll(tmp, 0750); err != nil {
			return fmt.Errorf("failed to create tmp directory: %w", err)
		}
		t, err := os.MkdirTemp(tmp, "log-")
		if err != nil {
			return err
		}
		dir = t
	} else {
		dir = filepath.Join(args[0], name)
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	kwokConfigPath := filepath.Join(dir, "kwok.yaml")
	if _, err := os.Stat(kwokConfigPath); err == nil {
		return fmt.Errorf("%s already exists", kwokConfigPath)
	}
	logger.Info("Exporting logs", "dir", dir)

	err = file.Copy(rt.GetWorkdirPath(runtime.ConfigName), kwokConfigPath)
	if err != nil {
		return err
	}

	if err = rt.CollectLogs(ctx, name, dir); err != nil {
		return err
	}

	return nil
}
