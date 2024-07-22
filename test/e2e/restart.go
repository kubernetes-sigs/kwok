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

package e2e

import (
	"context"
	"runtime"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseRestart is the restart of a test case.
func CaseRestart(kwokctlPath, clusterName string, clusterRuntime string, rootDir string, updateTestdata bool) *features.FeatureBuilder {
	node := helper.NewNodeBuilder("node0").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		Build()
	pod1 := helper.NewPodBuilder("pod1").
		Build()

	f0 := features.New("Restart").
		Setup(helper.CreateNode(node)).
		Setup(helper.CreatePod(pod0)).
		Teardown(helper.DeletePod(pod0)).
		Teardown(helper.DeleteNode(node)).
		Assess("test stop dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/stop_cluster.txt"
			args := []string{
				"stop", "cluster", "--name", clusterName, "--dry-run",
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
		}).
		Assess("test stop", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "stop", "cluster", "--name", clusterName)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("test start dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/start_cluster.txt"
			args := []string{
				"start", "cluster", "--name", clusterName, "--dry-run",
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
		}).
		Assess("test start", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "start", "cluster", "--name", clusterName)
			if err != nil {
				t.Fatal(err)
			}

			_, err = helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			_, err = helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}
			return ctx
		}).
		Assess("create pod1", helper.CreatePod(pod1)).
		Assess("delete pod1", helper.DeletePod(pod1))

	return f0
}
