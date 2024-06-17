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
	"path"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func getCurrentClusterDetails(clusterName string) (string, error) {
	cmd := exec.Command("kwokctl", "get", "cluster", clusterName, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	out := string(output[:])
	return out, err
}

func loadExpectedClusterDetails(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	out := string(data[:])
	return out, err
}

func CaseDryrun(clusterName string) *features.FeatureBuilder {
	f := features.New("Dry run")
	f = f.Assess("test cluster dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		got, err := getCurrentClusterDetails(clusterName)
		if err != nil {
			t.Fatal("Could not get cluster details:", err)
		}
		var expected string
		pwd := os.Getenv("PWD")
		rootDir := path.Join(pwd, "../../../..")
		expected, err = loadExpectedClusterDetails(rootDir + "test/kwokctl/testdata/binary/create_cluster_with_extra.txt")
		if err != nil {
			t.Fatal("Could not get expected cluster details:", err)
		}
		if !strings.EqualFold(strings.TrimSpace(got), strings.TrimSpace(expected)) {
			t.Fatalf("Expected %s but got %s", expected, got)
		}
		return ctx
	})
	return f
}
