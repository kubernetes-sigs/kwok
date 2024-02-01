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

// Package kind_test is a test environment for kwok.
package kind_test

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
	runtimeEnv  = consts.RuntimeTypeKind
	testEnv     env.Environment
	pwd         = os.Getenv("PWD")
	rootDir     = path.Join(pwd, "../../../..")
	logsDir     = path.Join(rootDir, "logs")
	clusterName = envconf.RandomName("kwok-e2e", 16)
	namespace   = envconf.RandomName("ns", 16)
	testImage   = "localhost/kwok:test"
	kwokctlPath = path.Join(rootDir, "bin", runtime.GOOS, runtime.GOARCH, "kwokctl"+helper.BinSuffix)
	baseArgs    = []string{
		"--kwok-controller-image=" + testImage,
		"--runtime=" + runtimeEnv,
		"--enable-metrics-server",
		"--wait=15m",
	}
)

func init() {
	_ = os.Setenv("KWOK_WORKDIR", path.Join(rootDir, "workdir"))
}

func TestMain(m *testing.M) {
	testEnv = helper.Environment()

	k := kwok.NewCluster(clusterName).
		WithPath(kwokctlPath)
	testEnv.Setup(
		helper.BuildKwokImage(rootDir, testImage, consts.RuntimeTypeDocker),
		helper.BuildKwokctlBinary(rootDir),
		helper.CreateCluster(k, append(baseArgs,
			"--config="+path.Join(rootDir, "test/e2e/port_forward.yaml"),
			"--config="+path.Join(rootDir, "test/e2e/logs.yaml"),
			"--config="+path.Join(rootDir, "test/e2e/attach.yaml"),
			"--config="+path.Join(rootDir, "test/e2e/exec.yaml"),
			"--config="+path.Join(rootDir, "kustomize/metrics/usage/usage-from-annotation.yaml"),
			"--config="+path.Join(rootDir, "kustomize/metrics/resource/metrics-resource.yaml"),
		)...),
		helper.CreateNamespace(namespace),
	)
	testEnv.Finish(
		helper.ExportLogs(k, logsDir),
		helper.DestroyCluster(k),
	)
	os.Exit(testEnv.Run(m))
}
