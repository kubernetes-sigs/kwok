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

// Package main implements a simple stage tester.
package main

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/tools/stage"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <resource file> <stage file...>\n", os.Args[0])
		os.Exit(1)
	}

	ctx := context.Background()
	err := run(ctx, args[0], args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, resourceFile string, stageFile []string) error {
	rio, err := config.LoadUnstructured(resourceFile)
	if err != nil {
		return err
	}
	if len(rio) != 1 {
		return fmt.Errorf("expected exactly one resource in %s, got %d", resourceFile, len(rio))
	}

	sio, err := config.Load(ctx, stageFile...)
	if err != nil {
		return err
	}

	ue := config.FilterWithoutType[*internalversion.Stage](sio)
	if len(ue) != 0 {
		return fmt.Errorf("expected only stage, got %d non-stage", len(ue))
	}

	stages := config.FilterWithType[*internalversion.Stage](sio)
	if len(stages) == 0 {
		return fmt.Errorf("expected at least one stage, got 0")
	}

	out, err := stage.TestingStages(ctx, rio[0], stages)
	if err != nil {
		return err
	}

	err = yaml.NewEncoder(os.Stdout).Encode(out)
	if err != nil {
		return err
	}

	return nil
}
