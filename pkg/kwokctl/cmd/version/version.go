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

// Package version implements the version command
package version

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/consts"
)

const cliName = "kwokctl"

// NewCommand returns a new cobra.Command for stop cluster
func NewCommand(ctx context.Context) *cobra.Command {
	var (
		short bool
	)
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "version",
		Short: "Print version information of kwokctl",
		Long:  "Print version information of kwokctl",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := consts.GetVersion()
			printClientVersion(&v, short)
			return nil
		},
	}
	cmd.Flags().BoolVar(&short, "short", false, "print just the version number")
	return cmd
}

func printClientVersion(version *consts.VersionInfo, short bool) {
	fmt.Printf("%s: %s\n", cliName, version)

	if short {
		return
	}

	fmt.Printf("  BuildDate: %s\n", version.BuildDate)
	fmt.Printf("  GoVersion: %s\n", version.GoVersion)
	fmt.Printf("  Compiler: %s\n", version.Compiler)
	fmt.Printf("  Platform: %s\n", version.Platform)
}
