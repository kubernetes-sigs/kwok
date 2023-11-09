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

// Package single_test is a test environment for kwok.
package single_test

import (
	"context"
	"os"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"

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

func TestMain(m *testing.M) {
	ctx := context.Background()
	cfg, _ := envconf.NewFromFlags()

	testEnv, _ = env.NewWithContext(ctx, cfg)
	deploy := pwd
	crs := path.Join(rootDir, "kustomize/stage/fast")
	testEnv.Setup(
		helper.BuildKwokImage(rootDir, testImage, consts.RuntimeTypeDocker),
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		helper.WaitForAllNodesReady(),
		envfuncs.LoadImageToCluster(clusterName, testImage),
		helper.CreateByKustomize(deploy),
		helper.WaitForAllPodsReady(),
		helper.CreateByKustomize(crs),
		helper.CreateNamespace(namespace),
	)
	testEnv.Finish(
		envfuncs.DestroyCluster(clusterName),
	)
	os.Exit(testEnv.Run(m))
}
