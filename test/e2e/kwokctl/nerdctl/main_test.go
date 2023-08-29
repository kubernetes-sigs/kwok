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

// Package nerdctl_test is a test environment for kwok.
package nerdctl_test

import (
	"context"
	"os"
	"runtime"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kwok"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

var (
	testEnv     env.Environment
	pwd         = os.Getenv("PWD")
	rootDir     = path.Join(pwd, "../../../..")
	clusterName = envconf.RandomName("kwok-e2e", 16)
	namespace   = envconf.RandomName("ns", 16)
	testImage   = "localhost/kwok:test"
)

func init() {
	_ = os.Setenv("KWOK_WORKDIR", path.Join(rootDir, "workdir"))
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, _ := envconf.NewFromFlags()

	testEnv, _ = env.NewWithContext(ctx, cfg)

	kwokctlPath := path.Join(rootDir, "bin", runtime.GOOS, runtime.GOARCH, "kwokctl"+helper.BinSuffix)

	k := kwok.NewProvider().
		WithName(clusterName).
		WithPath(kwokctlPath)
	testEnv.Setup(
		helper.BuildKwokImage(rootDir, testImage, consts.RuntimeTypeNerdctl),
		helper.BuildKwokctlBinary(rootDir),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			kubecfg, err := k.Create(ctx,
				"--kwok-controller-image", testImage,
				"--runtime", consts.RuntimeTypeNerdctl,
			)
			if err != nil {
				return ctx, err
			}

			cfg.WithKubeconfigFile(kubecfg)

			if err := k.WaitForControlPlane(ctx, cfg.Client()); err != nil {
				return ctx, err
			}

			return ctx, nil
		},
		envfuncs.CreateNamespace(namespace),
	)
	testEnv.Finish(
		func(ctx context.Context, config *envconf.Config) (context.Context, error) {
			err := k.Destroy(ctx)
			if err != nil {
				return ctx, err
			}

			return ctx, nil
		},
	)
	os.Exit(testEnv.Run(m))
}
