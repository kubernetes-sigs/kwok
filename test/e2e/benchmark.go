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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func waitDeleteResource(ctx context.Context, t *testing.T, kwokctlPath, name, resource string) error {
	for {
		data, err := exec.CommandContext(ctx, kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", resource).Output()
		if err != nil {
			return err
		}
		if len(data) == 0 {
			return nil
		}

		got := bytes.Count(data, []byte{'\n'})

		t.Logf("%s %d\n", resource, got)
		time.Sleep(1 * time.Second)
	}
}

func waitResource(ctx context.Context, t *testing.T, kwokctlPath, name, resource, reason string, want, gap, tolerance int, startFunc func() error) error {
	watchCtx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()
	cmd := exec.CommandContext(watchCtx, kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", "--watch", resource)
	cmd.Stderr = os.Stderr
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	if startFunc != nil {
		err = startFunc()
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}

	uniq := map[string]int{}
	prev := 0
	got := 0
	var latestTime time.Time
	reader := bufio.NewReader(pr)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			t.Logf("EOF: %s %d => %d, %v\n", resource, got, want, latestTime)

			cancel()

			time.Sleep(1 * time.Second)

			watchCtx, cancel = context.WithCancel(ctx) //nolint:govet
			cmd = exec.CommandContext(watchCtx, kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", "--watch", resource)
			cmd.Stderr = os.Stderr
			pr, err = cmd.StdoutPipe()
			if err != nil {
				return err //nolint:govet
			}

			err = cmd.Start()
			if err != nil {
				return err
			}

			reader = bufio.NewReader(pr)
			continue
		}

		key := string(line[:bytes.IndexByte(line, byte(' '))])

		_, ok := uniq[key]
		if !ok {
			uniq[key] = 0
		}

		if uniq[key] == 0 {
			if bytes.Contains(line, []byte(reason)) {
				uniq[key] = 1
				got++
			}
		}

		if got == want {
			t.Logf("%s %d, %v\n", resource, got, latestTime)
			return nil
		}

		if time.Since(latestTime) < time.Second {
			continue
		}

		latestTime = time.Now()

		all := len(uniq)

		t.Logf("%s %d/%d => %d, %v\n", resource, got, all, want, latestTime)
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
	}
}

func scaleCreatePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, size, gap, tolerance int) error {
	nodeName := "fake-node-000000"
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(size), "--param", fmt.Sprintf(".nodeName=%q", nodeName)) // #nosec G204
	scaleCmd.Stdout = os.Stderr
	scaleCmd.Stderr = os.Stderr

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "Running", size, gap, tolerance, scaleCmd.Start); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleDeletePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, _ int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(0)) // #nosec G204
	scaleCmd.Stdout = os.Stderr
	scaleCmd.Stderr = os.Stderr

	err := scaleCmd.Start()
	if err != nil {
		return err
	}

	if err := waitDeleteResource(ctx, t, kwokctlPath, name, "Pod"); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleCreateNode(ctx context.Context, t *testing.T, kwokctlPath string, name string, size, gap, tolerance int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "scale", "node", "fake-node", "--replicas", strconv.Itoa(size)) // #nosec G204
	scaleCmd.Stdout = os.Stderr
	scaleCmd.Stderr = os.Stderr

	if err := waitResource(ctx, t, kwokctlPath, name, "Node", "Ready", size, gap, tolerance, scaleCmd.Start); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func CaseBenchmark(kwokctlPath, clusterName string) *features.FeatureBuilder {
	return features.New("Benchmark Hack").
		Assess("Create nodes", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			err := scaleCreateNode(ctx0, t, kwokctlPath, clusterName, 5000, 5, 5)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Create pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			err := scaleCreatePod(ctx0, t, kwokctlPath, clusterName, 10000, 5, 5)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Delete pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			err := scaleDeletePod(ctx0, t, kwokctlPath, clusterName, 10000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
