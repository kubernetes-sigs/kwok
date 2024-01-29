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

// Package record provides a command to record the recording of a cluster.
package record

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

type flagpole struct {
	Name     string
	Path     string
	Prefix   string
	Snapshot bool
}

// NewCommand returns a new cobra.Command for cluster recording.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "record",
		Short: "Record the recording from the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix of the key")
	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the recording")
	cmd.Flags().BoolVar(&flags.Snapshot, "snapshot", false, "Only save the snapshot")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)
	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}
	if file.Exists(flags.Path) {
		return fmt.Errorf("file %q already exists", flags.Path)
	}

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

	etcdclient, err := rt.GetEtcdClient(ctx)
	if err != nil {
		return err
	}

	clientset, err := rt.GetClientset(ctx)
	if err != nil {
		return err
	}

	saver, err := etcd.NewSaver(etcd.SaveConfig{
		Clientset: clientset,
		Client:    etcdclient,
		Prefix:    flags.Prefix,
	})
	if err != nil {
		return err
	}

	f, err := file.Open(flags.Path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	var writer io.Writer = f

	startTime := time.Now()
	writer = recording.NewWriteHook(writer, func(bytes []byte) []byte {
		return recording.ReplaceTimeToRelative(startTime, bytes)
	})

	encoder := yaml.NewEncoder(writer)

	if flags.Snapshot {
		logger.Info("Saving snapshot")
	} else {
		logger.Info("Saving snapshot and recording")
	}

	err = saver.Save(ctx, encoder)
	if err != nil {
		return err
	}

	if flags.Snapshot {
		logger.Info("Saved snapshot")
		return nil
	}

	logger.Info("Recording")
	logger.Info("Press Ctrl+C to stop recording resources")

	err = saver.Record(ctx, encoder)
	if err != nil {
		return err
	}

	return nil
}
