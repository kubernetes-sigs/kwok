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
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func waitResource(ctx context.Context, t *testing.T, kwokctlPath, name, resource, reason string, want, gap, tolerance int) error {
	var prev int
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cmd := exec.Command(kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", resource) // #nosec G204
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		raw := string(output)
		got := strings.Count(raw, reason)
		if got == want {
			return nil
		}
		all := strings.Count(raw, "\n")
		t.Logf("%s %d/%d => %d\n", resource, got, all, want)
		if prev != 0 && got == prev {
			return fmt.Errorf("resource %s not changed", resource)
		}
		prev = got
		if gap != 0 && got != 0 && (all-got) > gap {
			if tolerance > 0 {
				t.Logf("Error %s gap too large, actual: %d, expected: %d, retrying...\n", resource, all-got, gap)
				tolerance--
			} else {
				t.Logf("Error %s gap too large, actual: %d, expected: %d\n", resource, all-got, gap)
				return fmt.Errorf("gap too large for resource %s", resource)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func scaleCreatePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	cmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "kubectl", "get", "node", "-o", "jsonpath={.items.*.metadata.name}") // #nosec G204
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}
	nodeName := ""
	nodes := strings.Split(string(out), " ")
	for _, node := range nodes {
		if strings.Contains(node, "fake-") {
			nodeName = node
			break
		}
	}
	if nodeName == "" {
		return fmt.Errorf("no fake- node found")
	}

	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(size), "--param", fmt.Sprintf(".nodeName=%q", nodeName)) // #nosec G204
	if err := scaleCmd.Start(); err != nil {
		return fmt.Errorf("failed to start scale command: %w", err)
	}

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "Running", size, 5, 1); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleDeletePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(size)) // #nosec G204
	if err := scaleCmd.Start(); err != nil {
		return fmt.Errorf("failed to start scale command: %w", err)
	}

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "fake-pod-", size, 0, 0); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleCreateNode(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "node", "fake-node", "--replicas", strconv.Itoa(size)) // #nosec G204
	if err := scaleCmd.Start(); err != nil {
		return fmt.Errorf("failed to start scale command: %w", err)
	}

	if err := waitResource(ctx, t, kwokctlPath, name, "Node", "Ready", size, 10, 5); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func CaseBenchmark(kwokctlPath, clusterName string) *features.FeatureBuilder {
	return features.New("Benchmark").
		Assess("Create nodes", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()

			err := scaleCreateNode(ctx0, t, kwokctlPath, clusterName, 5000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Create pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 240*time.Second)
			defer cancel()

			err := scaleCreatePod(ctx0, t, kwokctlPath, clusterName, 10000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Delete pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 240*time.Second)
			defer cancel()

			err := scaleDeletePod(ctx0, t, kwokctlPath, clusterName, 0)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
