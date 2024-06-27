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

// Package dryrun_test is a test environment for kwok.
package dryrun_test

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support/kwok"

	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

var (
	testEnv        env.Environment
	pwd            = os.Getenv("PWD")
	rootDir        = path.Join(pwd, "../../../..")
	logsDir        = path.Join(rootDir, "logs")
	clusterName    = envconf.RandomName("kwok-e2e-dryrun", 16)
	kwokctlPath    = path.Join(rootDir, "bin", runtime.GOOS, runtime.GOARCH, "kwokctl"+helper.BinSuffix)
	updateTestdata = false
)

func init() {
	_ = os.Setenv("KWOK_WORKDIR", path.Join(rootDir, "workdir"))
	flag.BoolVar(&updateTestdata, "update-testdata", false, "update all of testdata")
}

func TestMain(m *testing.M) {
	testEnv = helper.Environment()

	k := kwok.NewProvider().
		WithName(clusterName).
		WithPath(kwokctlPath)
	testEnv.Setup(
		helper.BuildKwokctlBinary(rootDir),
	)
	testEnv.Finish(
		helper.ExportLogs(k, logsDir),
	)
	os.Exit(testEnv.Run(m))
}
