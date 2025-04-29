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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func testWorkable(ctx context.Context, kwokctlPath, name string) error {
	cmd := exec.Command("kubectl", "config", "current-context")
	currentContext, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error getting current context: %w", err)
	}
	if strings.TrimSpace(string(currentContext)) != "kwok-"+name {
		return fmt.Errorf("current context is %s, expected kwok-%s", currentContext, name)
	}
	err = exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "node", "fake-node", "--replicas=1").Run()
	if err != nil {
		return fmt.Errorf("failed to scale node: %w", err)
	}
	err = exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas=1").Run()
	if err != nil {
		return fmt.Errorf("failed to scale pod: %w", err)
	}

	err = wait.For(
		func(ctx context.Context) (done bool, err error) {
			output, err := exec.CommandContext(ctx, kwokctlPath, "--name", name, "kubectl", "get", "pod").Output()
			if err != nil {
				return false, err
			}
			// TODO: Use json output instead and check that node and pod work as expected
			if !strings.Contains(string(output), "Running") {
				return false, fmt.Errorf("pod not running")
			}

			return true, nil
		},
		wait.WithContext(ctx),
		wait.WithTimeout(10*time.Second),
	)
	if err != nil {
		return err
	}

	return nil
}

func testGetKubeconfig(ctx context.Context, kwokctlPath, name string) error {
	kubeconfigPath := name + ".kubeconfig"
	output, err := exec.CommandContext(ctx, kwokctlPath, "--name", name, "get", "kubeconfig", "--user", "cluster-admin", "--group", "system:masters").Output()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	if err := os.WriteFile(kubeconfigPath, output, 0640); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	defer func() {
		_ = os.Remove(kubeconfigPath)
	}()
	output, err = exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigPath, "get", "pod").Output()
	if err != nil {
		return err
	}
	// TODO: Use json output instead and check that pod work as expected
	if !strings.Contains(string(output), "Running") {
		return fmt.Errorf("kubeconfig not working")
	}
	return nil
}

func testPrometheus() error {
	resp, err := http.Get("http://127.0.0.1:9090/api/v1/targets")
	if err != nil {
		return fmt.Errorf("failed to get targets: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// TODO: checks that the healthy component is consistent with the actual component being started
	if strings.Count(string(body), `"health":"up"`) < 6 {
		return fmt.Errorf("not enough healthy targets")
	}
	return nil
}

func testJaeger() error {
	resp, err := http.Get("http://127.0.0.1:16686/api/services")
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	// TODO: Check that the output is expected
	return nil
}

func testKwokControllerPort() error {
	resp, err := http.Get("http://127.0.0.1:10247/healthz")
	if err != nil {
		return fmt.Errorf("controller healthz check failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return fmt.Errorf("output is not unexpected: %q", string(body))
	}
	return nil
}

func testEtcdPort() error {
	resp, err := http.Get("http://127.0.0.1:2400/health")
	if err != nil {
		return fmt.Errorf("etcd health check failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), `"health"`) {
		return fmt.Errorf("output is not unexpected: %q", string(body))
	}
	return nil
}

func testEtcdctlGet(ctx context.Context, kwokctlPath, name string) error {
	output, err := exec.CommandContext(ctx, kwokctlPath, "--name="+name, "etcdctl", "get", "/registry/namespaces/default", "--keys-only").Output()
	if err != nil {
		return fmt.Errorf("failed to get namespace(default) by kwokctl etcdctl in cluster %s: %w", name, err)
	}
	if !strings.Contains(string(output), "default") {
		return fmt.Errorf("namespace(default) not found in cluster %s", name)
	}
	return nil
}

func testKubeSchedulerPort() error {
	// TODO: get CA from kwok cluster
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		},
	}
	resp, err := client.Get("https://127.0.0.1:10250/healthz")
	if err != nil {
		return fmt.Errorf("kube scheduler connection error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return fmt.Errorf("output is not unexpected: %q", string(body))
	}
	return nil
}

func testKubeControllerManagerPort() error {
	// TODO: get CA from kwok cluster
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		},
	}
	resp, err := client.Get("https://127.0.0.1:10260/healthz")
	if err != nil {
		return fmt.Errorf("kube controller manager connection error: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status is not 200 OK")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return fmt.Errorf("output is not unexpected: %q", string(body))
	}
	return nil
}

// CaseWorkable defines a feature test suite for verifying the basic functionality and workability of a KWOK cluster.
func CaseWorkable(kwokctlPath, clusterName, clusterRuntime string) *features.FeatureBuilder {
	f0 := features.New("Workable").
		Assess("Test Workable", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctxName := "kwok-" + clusterName
			err := exec.Command("kubectl", "config", "use-context", ctxName).Run()
			if err != nil {
				t.Fatal(err)
			}
			err = testWorkable(ctx, kwokctlPath, clusterName)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Test Get kubeconfig", func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			if err := testGetKubeconfig(ctx, kwokctlPath, clusterName); err != nil {
				t.Fatal(err)
			}
			return ctx
		})
	if clusterRuntime != "kind" && clusterRuntime != "kind-podman" {
		f0 = f0.
			Assess("Test Kube Controller Manager Port", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				if err := testKubeControllerManagerPort(); err != nil {
					t.Fatal(err)
				}
				return ctx
			}).
			Assess("Test Kube Scheduler Port", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				if err := testKubeSchedulerPort(); err != nil {
					t.Fatal(err)
				}
				return ctx
			}).
			Assess("Test Kube Etcdctl Port", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
				if err := testEtcdPort(); err != nil {
					t.Fatal(err)
				}
				return ctx
			})
	}
	if runtime.GOOS != "windows" {
		f0 = f0.Assess("Test Etcdctl Get", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if err := testEtcdctlGet(ctx, kwokctlPath, clusterName); err != nil {
				t.Fatal(err)
			}
			return ctx
		})
	}
	f0 = f0.
		Assess("Test Prometheus", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if err := testPrometheus(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Test Jaeger", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if err := testJaeger(); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Test Kwok controller port", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if err := testKwokControllerPort(); err != nil {
				t.Fatal(err)
			}
			return ctx
		})
	// TODO: Check that the other components work
	return f0
}
