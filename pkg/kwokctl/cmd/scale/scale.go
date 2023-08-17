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

// Package scale contains a command to scale a resource in a cluster.
package scale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/kustomize/kwokctl/resource"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/scale"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type flagpole struct {
	Name string

	SerialLength int
	Namespace    string
	Replicas     uint64
	Params       []string
}

// NewCommand returns a new cobra.Command for scale resource.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(1, 2),
		Use:   "scale [node, pod, ...] [name]",
		Short: "Scale a resource in cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags, args)
		},
	}
	cmd.Flags().Uint64Var(&flags.Replicas, "replicas", 1, "Number of replicas")
	cmd.Flags().IntVar(&flags.SerialLength, "serial-length", 6, "Length of serial number")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", flags.Namespace, "Namespace of resource to scale")
	cmd.Flags().StringArrayVar(&flags.Params, "param", flags.Params, "Parameter to update")
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

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
	resourceKind := args[0]
	resourceName := resourceKind
	if len(args) == 2 {
		resourceName = args[1]
	}

	kubeconfigPath := rt.GetWorkdirPath(runtime.InHostKubeconfigName)
	clientset, err := client.NewClientset("", kubeconfigPath)
	if err != nil {
		return err
	}

	krcs := config.FilterWithTypeFromContext[*internalversion.KwokctlResource](ctx)
	krc, ok := slices.Find(krcs, func(krc *internalversion.KwokctlResource) bool {
		return krc.Name == resourceKind
	})
	if !ok {
		var resourceData string
		switch resourceKind {
		default:
			return fmt.Errorf("resource %s is not exists", resourceKind)
		case "pod":
			resourceData = resource.DefaultPod
		case "node":
			resourceData = resource.DefaultNode
		}

		logger.Info("No resource found, use default resource", "resource", resourceKind)
		krc, err = config.UnmarshalWithType[*internalversion.KwokctlResource](resourceData)
		if err != nil {
			return err
		}
	}

	parameters, err := scale.NewParameters(ctx, krc.Parameters, flags.Params)
	if err != nil {
		return err
	}

	err = scale.Scale(ctx, clientset, scale.Config{
		Parameters:   parameters,
		Template:     krc.Template,
		Name:         resourceName,
		Namespace:    flags.Namespace,
		Replicas:     int(flags.Replicas),
		SerialLength: flags.SerialLength,
		DryRun:       dryrun.DryRun,
	})
	if err != nil {
		return err
	}
	return nil
}
