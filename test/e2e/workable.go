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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func testWorkable(kwokctlPath, name string) error {
	cmd := exec.Command("kubectl", "config", "current-context")
	currentContext, err := cmd.Output()
	if err != nil {
		fmt.Print("THIS IS THE ERROR:", err)
		return fmt.Errorf("error getting current context: %v", err)
	}
	if strings.TrimSpace(string(currentContext)) != "kwok-"+name {
		return fmt.Errorf("current context is %s, expected kwok-%s", currentContext, name)
	}
	if err := retry(120, func() error {
		return exec.Command(kwokctlPath, "--name", name, "scale", "node", "fake-node", "--replicas=1").Run()
	}); err != nil {
		return fmt.Errorf("failed to scale node: %v", err)
	}
	if err := retry(120, func() error {
		return exec.Command(kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas=1").Run()
	}); err != nil {
		return fmt.Errorf("failed to scale pod: %v", err)
	}
	if err := retry(120, func() error {
		output, err := exec.Command(kwokctlPath, "--name", name, "kubectl", "get", "pod").Output()
		if err != nil {
			return err
		}
		if !strings.Contains(string(output), "Running") {
			return fmt.Errorf("pod not running")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("cluster not ready: %v", err)
	}

	kubeconfigPath := name + ".kubeconfig"
	output, err := exec.Command(kwokctlPath, "--name", name, "get", "kubeconfig", "--user", "cluster-admin", "--group", "system:masters").Output()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	if err := os.WriteFile(kubeconfigPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %v", err)
	}

	if err := retry(120, func() error {
		output, err := exec.Command("kubectl", "--kubeconfig", kubeconfigPath, "get", "pod").Output()
		if err != nil {
			return err
		}
		if !strings.Contains(string(output), "Running") {
			return fmt.Errorf("kubeconfig not working")
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func testPrometheus() error {
	var targets string
	if err := retry(120, func() error {
		resp, err := http.Get("http://127.0.0.1:9090/api/v1/targets")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		targets = string(body)
		if strings.Count(targets, `"health":"up"`) < 6 {
			return fmt.Errorf("not enough healthy targets")
		}
		return nil
	}); err != nil {
		fmt.Println("Error: metrics is not healthy")
		fmt.Println("curl -s http://127.0.0.1:9090/api/v1/targets")
		fmt.Println(targets)
		return err
	}

	return nil
}

func testJaeger() error {
	return retry(120, func() error {
		resp, err := http.Get("http://127.0.0.1:16686/api/services")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		return fmt.Errorf("jaeger not healthy")
	})
}

func testKwokControllerPort() error {
	resp, err := http.Get("http://127.0.0.1:10247/healthz")
	if err != nil {
		return fmt.Errorf("controller healthz check failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "ok" {
		fmt.Println("Error: controller healthz is not ok")
		fmt.Println("curl -s http://127.0.0.1:10247/healthz")
		fmt.Println(string(body))
		return fmt.Errorf("controller healthz is not ok")
	}

	return nil
}

func testEtcdPort() error {
	resp, err := http.Get("http://127.0.0.1:2400/health")
	if err != nil {
		return fmt.Errorf("etcd health check failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if !strings.Contains(string(body), `"health"`) {
		fmt.Println("Error: etcd connection")
		fmt.Println("curl -s http://127.0.0.1:2400/health")
		fmt.Println(string(body))
		return fmt.Errorf("etcd connection error")
	}

	return nil
}

func testEtcdctlGet(kwokctlPath, name string) error {
	output, err := exec.Command(kwokctlPath, "--name="+name, "etcdctl", "get", "/registry/namespaces/default", "--keys-only").Output()
	if err != nil {
		return fmt.Errorf("failed to get namespace(default) by kwokctl etcdctl in cluster %s: %v", name, err)
	}

	if !strings.Contains(string(output), "default") {
		return fmt.Errorf("namespace(default) not found in cluster %s", name)
	}

	return nil
}

func testKubeSchedulerPort() error {
	proto := "https"

	resp, err := http.Get(fmt.Sprintf("%s://127.0.0.1:10250/healthz", proto))
	if err != nil {
		return fmt.Errorf("kube scheduler connection error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "ok" {
		fmt.Println("Error: kube scheduler connection")
		fmt.Println("curl -s " + proto + "://127.0.0.1:10250/healthz")
		fmt.Println(string(body))
		return fmt.Errorf("kube scheduler connection error")
	}

	return nil
}

func testKubeControllerManagerPort() error {
	proto := "https"

	resp, err := http.Get(fmt.Sprintf("%s://127.0.0.1:10260/healthz", proto))
	if err != nil {
		return fmt.Errorf("kube controller manager connection error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "ok" {
		fmt.Println("Error: kube controller manager connection")
		fmt.Println("curl -s " + proto + "://127.0.0.1:10260/healthz")
		fmt.Println(string(body))
		return fmt.Errorf("kube controller manager connection error")
	}

	return nil
}

func retry(times int, f func() error) error {
	for i := 0; i < times; i++ {
		if err := f(); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("retry failed after %d attempts", times)
}

func CaseWorkable(kwokctlPath, clusterName, clusterRuntime string) *features.FeatureBuilder {
	return features.New("Workable").
		Assess("Test Workable", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			_, err := exec.CommandContext(ctx, "kubectl", "config", "current-context").Output()
			if err != nil {
				fmt.Print("ERRRRRRR")
				t.Fatal(err)
			}
			err = testWorkable(kwokctlPath, clusterName)
			if err != nil {
				t.Fatal(err)
			}
			if clusterRuntime != "kind" && clusterRuntime != "kind-podman" {
				if err = testKubeControllerManagerPort(); err != nil {
					t.Fatal(err)
				}
				if err = testKubeSchedulerPort(); err != nil {
					t.Fatal(err)
				}
				if err = testEtcdPort(); err != nil {
					t.Fatal(err)
				}
			}
			if runtime.GOOS != "windows" {
				if err = testEtcdctlGet(kwokctlPath, clusterName); err != nil {
					t.Fatal(err)
				}
			}
			if err = testPrometheus(); err != nil {
				t.Fatal(err)
			}
			if err = testJaeger(); err != nil {
				t.Fatal(err)
			}
			if err = testKwokControllerPort(); err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
