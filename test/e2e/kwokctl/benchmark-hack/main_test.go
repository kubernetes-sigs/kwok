/*
Copyright 2025 The Kubernetes Authors.

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

// Package benchmark_hack_test is a test benchmarking environment for kwok.
package benchmark_hack_test

import (
	"os"
	"runtime"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support/kwok"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

var (
	runtimeEnv  = consts.RuntimeTypeBinary
	testEnv     env.Environment
	pwd         = os.Getenv("PWD")
	rootDir     = path.Join(pwd, "../../../..")
	logsDir     = path.Join(rootDir, "logs")
	clusterName = envconf.RandomName("kwok-e2e-benchmark-hack", 30)
	kwokPath    = path.Join(rootDir, "bin", runtime.GOOS, runtime.GOARCH, "kwok"+helper.BinSuffix)
	kwokctlPath = path.Join(rootDir, "bin", runtime.GOOS, runtime.GOARCH, "kwokctl"+helper.BinSuffix)
	baseArgs    = []string{
		"--kwok-controller-binary=" + kwokPath,
		"--runtime=" + runtimeEnv,
		"--wait=15m",
		"--disable-kube-scheduler",
		"--disable-qps-limits",
	}
)

func init() {
	_ = os.Setenv("KWOK_WORKDIR", path.Join(rootDir, "workdir"))
}

func TestMain(m *testing.M) {
	testEnv = helper.Environment()

	k := kwok.NewProvider().
		WithName(clusterName).
		WithPath(kwokctlPath)
	testEnv.Setup(
		helper.BuildKwokBinary(rootDir),
		helper.BuildKwokctlBinary(rootDir),
		helper.CreateCluster(k, baseArgs...),
	)
	testEnv.Finish(
		helper.ExportLogs(k, logsDir),
		helper.DestroyCluster(k),
	)
	os.Exit(testEnv.Run(m))
}
