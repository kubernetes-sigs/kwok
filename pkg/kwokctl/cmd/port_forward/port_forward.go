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

// Package port_forward implements the `port-forward` command
package port_forward

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
)

type flagpole struct {
	Name string
}

// NewCommand returns a new cobra.Command for forwarding the port
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.ExactArgs(2),
		Use:   "port-forward [component] [local-port]:[port-name]",
		Short: "Forward one local ports to a component",
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
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	hostPort, containerPort, err := splitParts(args[1])
	if err != nil {
		return err
	}

	port, err := strconv.ParseUint(hostPort, 0, 0)
	if err != nil {
		return err
	}

	cancel, err := rt.PortForward(ctx, args[0], containerPort, uint32(port))
	if err != nil {
		return err
	}

	defer cancel()
	<-ctx.Done()

	return nil
}

func splitParts(rawport string) (hostPort string, containerPort string, err error) {
	parts := strings.Split(rawport, ":")
	n := len(parts)

	switch n {
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("unsupported port format: %s", rawport)
	}
}
