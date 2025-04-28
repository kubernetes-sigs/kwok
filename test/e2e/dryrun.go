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
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/path"
)

var (
	emptyLine  = regexp.MustCompile("\n{2,}")
	homeDir, _ = os.UserHomeDir()
)

func formatCmdOutput(got, clusterName, rootDir string) string {
	got = strings.ReplaceAll(got, clusterName, "<CLUSTER_NAME>")
	got = strings.ReplaceAll(got, rootDir, "<ROOT_DIR>")
	got = strings.ReplaceAll(got, runtime.GOOS, "<OS>")
	got = strings.ReplaceAll(got, runtime.GOARCH, "<ARCH>")
	got = strings.ReplaceAll(got, ".zip", ".<TAR>")
	got = strings.ReplaceAll(got, ".tar.gz", ".<TAR>")
	got = strings.ReplaceAll(got, homeDir, "~")
	got = strings.ReplaceAll(got, "/root", "~")
	got = strings.ReplaceAll(got, " --env=ETCD_UNSUPPORTED_ARCH=<ARCH> ", " ")
	got = strings.ReplaceAll(got, " ETCD_UNSUPPORTED_ARCH=<ARCH> ", " ")
	got = emptyLine.ReplaceAllLiteralString(got, "\n")
	return got
}

func executeCommand(args []string, absPath string, clusterName string, kwokctlPath string, rootDir string, updateTestdata bool) (string, error) {
	testdataPath := path.Join(rootDir, absPath)
	expected, err := os.ReadFile(testdataPath)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(kwokctlPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	want := string(expected)
	got := string(output)
	got = formatCmdOutput(got, clusterName, rootDir)
	if diff := cmp.Diff(got, want); diff != "" {
		if updateTestdata {
			err = os.WriteFile(testdataPath, []byte(got), fs.FileMode(0644))
			if err != nil {
				return "", err
			}
		} else {
			return diff, nil
		}
	}
	return "", nil
}

// CaseDryrun defines a feature test suite for verifying the dry-run functionality of cluster creation.
// It tests the output of kwokctl create cluster --dry-run command against expected test data.
// The test ensures the dry-run output matches the expected format and contents.
func CaseDryrun(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run")
	f = f.Assess("test cluster dryrun", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster.txt"
		args := []string{
			"create", "cluster", "--dry-run",
			"--name", clusterName,
			"--timeout=30m",
			"--wait=30m",
			"--quiet-pull",
			"--disable-qps-limits",
			"--kube-authorization=false",
			"--runtime", clusterRuntime,
		}
		diff, err := executeCommand(args, absPath, clusterName, kwokctlPath, rootDir, updateTestdata)
		if err != nil {
			t.Fatal(err)
		}
		if diff != "" {
			t.Fatalf("Expected vs got:\n%s", diff)
		}
		return ctx
	})
	return f
}

// CaseDryrunWithExtra defines a feature test suite for verifying the dry-run functionality of cluster creation
// with additional configuration. It tests the output of kwokctl create cluster --dry-run --config <extra-config>
// command against expected test data. The test ensures the dry-run output with extra configuration matches
// the expected format and contents.
func CaseDryrunWithExtra(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run with extra")
	f = f.Assess("test cluster dryrun with extra", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster_with_extra.txt"
		extraPath := path.Join(rootDir, "test/e2e/kwokctl/dryrun/testdata/extra.yaml")
		args := []string{
			"create", "cluster", "--dry-run",
			"--runtime", clusterRuntime,
			"--name", clusterName,
			"--config", extraPath,
			"--timeout=30m",
			"--wait=30m",
			"--quiet-pull",
			"--disable-qps-limits",
		}
		diff, err := executeCommand(args, absPath, clusterName, kwokctlPath, rootDir, updateTestdata)
		if err != nil {
			t.Fatal(err)
		}
		if diff != "" {
			t.Fatalf("Expected vs got:\n%s", diff)
		}
		return ctx
	})
	return f
}

// CaseDryrunWithVerbosity defines a feature test suite for verifying the dry-run functionality of cluster creation
// with verbose configuration options. It tests the output of kwokctl create cluster --dry-run command with various
// verbose flags against expected test data. The test ensures the dry-run output with verbose configuration matches
// the expected format and contents, including ports for monitoring components and custom configuration paths.
func CaseDryrunWithVerbosity(clusterName string, kwokctlPath string, rootDir string, clusterRuntime string, updateTestdata bool) *features.FeatureBuilder {
	f := features.New("Dry run with verbosity")
	f = f.Assess("test cluster dryrun with verbosity", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		absPath := "test/e2e/kwokctl/dryrun/testdata/" + clusterRuntime + "/create_cluster_with_verbosity.txt"
		kubeAuditPath := path.Join(rootDir, "test/kwokctl/audit-policy.yaml")
		schedulerConfigPath := path.Join(rootDir, "test/kwokctl/scheduler-config.yaml")
		args := []string{
			"create", "cluster", "--dry-run",
			"--runtime", clusterRuntime,
			"--name", clusterName,
			"--timeout=30m",
			"--wait=30m",
			"--quiet-pull",
			"--disable-qps-limits",
			"--prometheus-port=9090",
			"--jaeger-port=16686",
			"--dashboard-port=8000",
			"--kube-apiserver-insecure-port=6080",
			"--enable-metrics-server",
			"--kube-audit-policy", kubeAuditPath,
			"--kube-scheduler-config", schedulerConfigPath,
		}
		diff, err := executeCommand(args, absPath, clusterName, kwokctlPath, rootDir, updateTestdata)
		if err != nil {
			t.Fatal(err)
		}
		if diff != "" {
			t.Fatalf("Expected vs got:\n%s", diff)
		}
		return ctx
	})
	return f
}
