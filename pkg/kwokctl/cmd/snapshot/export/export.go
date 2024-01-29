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
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/recording"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

type flagpole struct {
	Path              string
	Kubeconfig        string
	Filters           []string
	ImpersonateUser   string
	ImpersonateGroups []string
	PageSize          int64
	PageBufferSize    int32
	Record            bool
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
	cmd.Flags().BoolVar(&flags.Record, "record", false, "Record the change of the cluster")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	if flags.Path == "" {
		return fmt.Errorf("path is required")
	}
	if file.Exists(flags.Path) {
		return fmt.Errorf("file %q already exists", flags.Path)
	}

	if dryrun.DryRun {
		dryrun.PrintMessage("kubectl --kubeconfig %s get %s -o yaml >%s", flags.Kubeconfig, strings.Join(flags.Filters, ","), flags.Path)
		return nil
	}

	clientset, err := client.NewClientset("", flags.Kubeconfig,
		client.WithImpersonate(rest.ImpersonationConfig{
			UserName: flags.ImpersonateUser,
			Groups:   flags.ImpersonateGroups,
		}),
	)
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

	logger := log.FromContext(ctx)

	pagerConfig := &snapshot.PagerConfig{
		PageSize:       flags.PageSize,
		PageBufferSize: flags.PageBufferSize,
	}

	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return err
	}

	filters, errs := client.MappingForResources(restMapper, flags.Filters)
	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error("failed to get mapping", err)
		}
	}

	saver, err := snapshot.NewSaver(snapshot.SaveConfig{
		Clientset:   clientset,
		PagerConfig: pagerConfig,
		Filters:     filters,
	})
	if err != nil {
		return err
	}

	var writer io.Writer = f

	startTime := time.Now()
	writer = recording.NewWriteHook(writer, func(b []byte) []byte {
		return recording.ReplaceTimeToRelative(startTime, b)
	})

	encoder := yaml.NewEncoder(writer)

	var tracks map[*meta.RESTMapping]*snapshot.TrackData
	if flags.Record {
		tracks = make(map[*meta.RESTMapping]*snapshot.TrackData)
	}
	err = saver.Save(ctx, encoder, tracks)
	if err != nil {
		return err
	}

	if flags.Record {
		logger.Info("Recording")
		logger.Info("Press Ctrl+C to stop recording resources")

		err = saver.Record(ctx, encoder, tracks)
		if err != nil {
			return err
		}
	}

	return nil
}
