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
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	out = strings.ReplaceAll(out, "\n", " ")
	out = strings.Join(strings.Fields(out), " ")
	return out, nil
}

func formatCmdOutput(output, clusterName, rootDir string) string {
	got := output
	extensions := map[string]string{
		"windows": "zip",
		"linux":   "tar.gz",
		"darwin":  "tar.gz",
	}
	got = strings.ReplaceAll(got, clusterName, "<CLUSTER_NAME>")
	got = strings.ReplaceAll(got, rootDir, "<ROOT_DIR>")
	got = strings.ReplaceAll(got, runtime.GOOS, "<OS>")
	got = strings.ReplaceAll(got, runtime.GOARCH, "<ARCH>")
	got = strings.ReplaceAll(got, extensions[runtime.GOOS], "<TAR>")
	got = strings.ReplaceAll(got, "\n", " ")
	got = strings.Join(strings.Fields(got), " ")
	return got
}

func CaseDryrun(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run")
	f = f.Assess("test cluster dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		var expected string
		var err error
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster.txt"
		expected, err = loadExpectedClusterDetails(path.Join(rootDir, absPath))
		if err != nil {
			t.Fatal("Could not get expected cluster details:", err)
		}
		args := []string{
			"create", "cluster", "--dry-run", "--name", clusterName, "--timeout=30m",
			"--wait=30m", "--quiet-pull", "--disable-qps-limits", "--kube-authorization=false",
			"--runtime", clusterRuntime,
		}
		cmd := exec.Command(kwokctlPath, args...) // #nosec G204
		var output []byte
		output, err = cmd.Output()
		if err != nil {
			t.Fatal("Could not get the output of the command:", err)
		}
		got := string(output)
		got = formatCmdOutput(got, clusterName, rootDir)
		if diff := cmp.Diff(strings.TrimSpace(got), strings.TrimSpace(expected)); diff != "" {
			if updateTestdata {
				err = os.WriteFile(path.Join(rootDir, absPath), output, fs.FileMode(0644))
				if err != nil {
					t.Fatal("Could not write file:", err)
				}
			} else {
				t.Fatalf("Expected vs got:\n%s", diff)
			}
		}
		return ctx
	})
	return f
}

func CaseDryrunWithExtra(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run with extra")
	f = f.Assess("test cluster dryrun with extra", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		var expected string
		var err error
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster_with_extra.txt"
		expected, err = loadExpectedClusterDetails(path.Join(rootDir, absPath))
		if err != nil {
			t.Fatal("Could not get expected cluster details:", err)
		}
		extraPath := path.Join(rootDir, "test/kwokctl/testdata/extra.yaml")
		args := []string{
			"create", "cluster", "--dry-run", "--name", clusterName, "--timeout=30m",
			"--wait=30m", "--quiet-pull", "--disable-qps-limits", "--runtime", clusterRuntime,
			"--config", extraPath,
		}
		cmd := exec.Command(kwokctlPath, args...) // #nosec G204
		var output []byte
		output, err = cmd.Output()
		if err != nil {
			t.Fatal("Could not get the output of the command:", err)
		}
		got := string(output)
		got = formatCmdOutput(got, clusterName, rootDir)
		if diff := cmp.Diff(strings.TrimSpace(got), strings.TrimSpace(expected)); diff != "" {
			if updateTestdata {
				err = os.WriteFile(path.Join(rootDir, absPath), output, fs.FileMode(0644))
				if err != nil {
					t.Fatal("Could not write file:", err)
				}
			} else {
				t.Fatalf("Expected vs got:\n%s", diff)
			}
		}
		return ctx
	})
	return f
}

func CaseDryrunWithVerbosity(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run with verbosity")
	f = f.Assess("test cluster dryrun with verbosity", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		var expected string
		var err error
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster_with_verbosity.txt"
		expected, err = loadExpectedClusterDetails(path.Join(rootDir, absPath))
		if err != nil {
			t.Fatal("Could not get expected cluster details:", err)
		}
		kubeAuditPath := path.Join(rootDir, "test/kwokctl/audit-policy.yaml")
		schedulerConfigPath := path.Join(rootDir, "test/kwokctl/scheduler-config.yaml")
		args := []string{
			"create", "cluster", "--dry-run", "--name", clusterName, "--timeout=30m", "--wait=30m",
			"--quiet-pull", "--disable-qps-limits", "--runtime", clusterRuntime,
			"--prometheus-port=9090", "--jaeger-port=16686", "--dashboard-port=8000",
			"--enable-metrics-server", "--kube-audit-policy", kubeAuditPath,
			"--kube-scheduler-config", schedulerConfigPath,
		}
		cmd := exec.Command(kwokctlPath, args...) // #nosec G204
		var output []byte
		output, err = cmd.Output()
		if err != nil {
			t.Fatal("Could not get the output of the command:", err)
		}
		got := string(output)
		got = formatCmdOutput(got, clusterName, rootDir)
		if diff := cmp.Diff(strings.TrimSpace(got), strings.TrimSpace(expected)); diff != "" {
			if updateTestdata {
				err = os.WriteFile(path.Join(rootDir, absPath), output, fs.FileMode(0644))
				if err != nil {
					t.Fatal("Could not write file:", err)
				}
			} else {
				t.Fatalf("Expected vs got:\n%s", diff)
			}
		}
		return ctx
	})
	return f
}
