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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kwok/pkg/config"
	kwokcmd "sigs.k8s.io/kwok/pkg/kwok/cmd"
	kwokctlcmd "sigs.k8s.io/kwok/pkg/kwokctl/cmd"
	"sigs.k8s.io/kwok/pkg/log"

	_ "sigs.k8s.io/kwok/pkg/kwokctl/runtime/binary"
	_ "sigs.k8s.io/kwok/pkg/kwokctl/runtime/compose"
	_ "sigs.k8s.io/kwok/pkg/kwokctl/runtime/kind"
)

const basePath = "./site/content/en/docs/generated/"

func main() {
	if err := os.MkdirAll(basePath, os.FileMode(0755)); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	config.GOOS = "linux"
	config.GOARCH = "amd64"

	flagset := pflag.NewFlagSet("global", pflag.ContinueOnError)
	flagset.ParseErrorsWhitelist.UnknownFlags = true
	flagset.Usage = func() {}
	ctx := context.Background()
	ctx, logger := log.InitFlags(ctx, flagset)

	_, err := config.InitFlags(ctx, flagset)
	if err != nil {
		_, _ = os.Stderr.Write([]byte(flagset.FlagUsages()))
		logger.Error("Init config flags", err)
		os.Exit(1)
	}
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.WarnLevel))

	genKwok(ctx, flagset, basePath)
	genKwokctl(ctx, flagset, basePath)
}

func genKwok(ctx context.Context, flags *pflag.FlagSet, basePath string) {
	rootCmd := kwokcmd.NewCommand(ctx)
	rootCmd.PersistentFlags().AddFlagSet(flags)
	rootCmd.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(rootCmd, basePath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func genKwokctl(ctx context.Context, flags *pflag.FlagSet, basePath string) {
	rootCmd := kwokctlcmd.NewCommand(ctx)
	rootCmd.PersistentFlags().AddFlagSet(flags)
	rootCmd.DisableAutoGenTag = true
	if err := doc.GenMarkdownTree(rootCmd, basePath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
