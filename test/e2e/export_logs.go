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

package e2e

import (
	"context"
	"runtime"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func CaseDryRunExportLogs(kwokctlPath, clusterName, clusterRuntime string, rootDir string, updateTestdata bool) *features.FeatureBuilder {
	return features.New("Dryrun Export Logs").
		Assess("test dryrun export logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/export_logs.txt"
			args := []string{
				"--name", clusterName, "--dry-run", "export", "logs",
			}
			diff, err := executeCommand(args, absPath, clusterName, kwokctlPath, rootDir, updateTestdata)
			if err != nil {
				t.Fatal(err)
			}
			if diff != "" && runtime.GOOS == "linux" {
				updateCmd := "go test -v ./test/e2e/kwokctl/" + clusterRuntime + " -args --update-testdata"
				t.Fatalf("Expected vs got:\n%s\nExeceute this command to update the testdata manually:%s", diff, updateCmd)
			}
			return ctx
		})
}
