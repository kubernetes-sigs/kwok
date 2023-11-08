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

package helper

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"sigs.k8s.io/kwok/pkg/utils/exec"
)

// BuildKwokImage builds the kwok image and returns a function that can be used
func BuildKwokImage(rootDir string, image string, builder string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		ref := strings.SplitN(image, ":", 2)
		if len(ref) != 2 {
			return nil, fmt.Errorf("invalid image reference %q", image)
		}
		ctx = exec.WithStdIO(ctx)
		ctx = exec.WithDir(ctx, rootDir)

		err := exec.Exec(ctx, "bash", "./hack/releases.sh", "--bin", "kwok", "--platform", "linux/"+runtime.GOARCH)
		if err != nil {
			return nil, err
		}

		err = exec.Exec(ctx, "bash", "./images/kwok/build.sh", "--image", ref[0], "--builder", builder, "--version", ref[1], "--platform", "linux/"+runtime.GOARCH)
		if err != nil {
			return nil, err
		}
		return ctx, nil
	}
}

// BuildKwokBinary builds the kwok binary and returns a function that can be used
func BuildKwokBinary(rootDir string) env.Func {
	return buildBinary(rootDir, "kwok")
}

// BuildKwokctlBinary builds the kwokctl binary and returns a function that can be used
func BuildKwokctlBinary(rootDir string) env.Func {
	return buildBinary(rootDir, "kwokctl")
}

// buildBinary builds the kwok binary and returns a function that can be used
func buildBinary(rootDir string, binary string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		ctx = exec.WithStdIO(ctx)
		ctx = exec.WithDir(ctx, rootDir)

		err := exec.Exec(ctx, "bash", "./hack/releases.sh", "--bin", binary, "--platform", runtime.GOOS+"/"+runtime.GOARCH)
		if err != nil {
			return nil, err
		}
		return ctx, nil
	}
}
