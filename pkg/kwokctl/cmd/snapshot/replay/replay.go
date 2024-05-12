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

// Package replay provides a command to replay the recordingof a cluster.
package replay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

type flagpole struct {
	Name     string
	Path     string
	Snapshot bool
}

// NewCommand returns a new cobra.Command to replay the cluster as a recording.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "replay",
		Short: "Replay the recording to the cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringVar(&flags.Path, "path", "", "Path to the recording")
	cmd.Flags().BoolVar(&flags.Snapshot, "snapshot", false, "Only restore the snapshot")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)
	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}
	if !file.Exists(flags.Path) {
		return fmt.Errorf("path %q does not exist", flags.Path)
	}

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	conf, err := rt.Config(ctx)
	if err != nil {
		return err
	}

	components, err := rt.ListComponents(ctx)
	if err != nil {
		return err
	}

	components = slices.Filter(components, func(component internalversion.Component) bool {
		return component.Name != consts.ComponentKubeApiserver && component.Name != consts.ComponentEtcd
	})

	for _, component := range components {
		err = rt.StopComponent(ctx, component.Name)
		if err != nil {
			logger.Error("Failed to stop component", err,
				"component", component.Name,
			)
		}
	}

	defer func() {
		for _, component := range components {
			err = rt.StartComponent(ctx, component.Name)
			if err != nil {
				logger.Error("Failed to start component", err,
					"component", component.Name,
				)
			}
		}
	}()

	etcdclient, err := rt.GetEtcdClient(ctx)
	if err != nil {
		return err
	}

	clientset, err := rt.GetClientset(ctx)
	if err != nil {
		return err
	}

	loader, err := etcd.NewLoader(etcd.LoadConfig{
		Clientset: clientset,
		Client:    etcdclient,
		Prefix:    conf.Options.EtcdPrefix,
	})
	if err != nil {
		return err
	}

	f, err := os.Open(flags.Path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	press, err := file.Decompress(flags.Path, f)
	if err != nil {
		return err
	}
	defer func() {
		_ = press.Close()
	}()

	var reader io.Reader = press

	startTime := time.Now()
	reader = recording.NewReadHook(reader, func(bytes []byte) []byte {
		return recording.RevertTimeFromRelative(startTime, bytes)
	})

	decoder := yaml.NewDecoder(reader)

	if flags.Snapshot {
		logger.Info("Restoring snapshot")
	} else {
		logger.Info("Restoring snapshot and replaying")
	}

	err = loader.Load(ctx, decoder)
	if err != nil {
		return err
	}

	if flags.Snapshot {
		logger.Info("Restored snapshot")
		return nil
	}

	logger.Info("Replaying")
	if log.IsTerminal() {
		cancel := loader.AllowHandle(ctx)
		defer cancel()
	}
	err = loader.Replay(ctx, decoder)
	if err != nil {
		return err
	}

	return nil
}
