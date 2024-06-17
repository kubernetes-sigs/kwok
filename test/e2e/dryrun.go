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
	"os"
	"os/exec"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/path"
)

func loadExpectedClusterDetails(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	out := string(data)
	return out, nil
}

func CaseDryrun(clusterName string, kwokctlPath string, rootDir string) *features.FeatureBuilder {
	f := features.New("Dry run")
	f = f.Assess("test cluster dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		var expected string
		var err error
		expected, err = loadExpectedClusterDetails(path.Join(rootDir, "test/kwokctl/testdata/binary/create_cluster.txt"))
		if err != nil {
			t.Fatal("Could not get expected cluster details:", err)
		}
		t.Log("EXPECTED CLUSTER:", expected)
		cmd := exec.Command(kwokctlPath, "create", "cluster", "--name", clusterName, "--dry-run", "--runtime=binary", "--kube-authorization=false", "--disable-qps-limits", "--quiet-pull", "--timeout=30m", "--wait=30m")
		var output []byte
		output, err = cmd.Output()
		if err != nil {
			t.Fatal("Could not get the output of the command:", err)
		}
		got := string(output)
		got = strings.ReplaceAll(got, clusterName, "<CLUSTER_NAME>")
		got = strings.ReplaceAll(got, rootDir, "<ROOT_DIR>")
		t.Log("GOT CLUSTER:", got)
		if !strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(expected)) {
			t.Fatalf("Expected %s but got %s", expected, got)
		}
		return ctx
	})
	return f
}
