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
	"os/exec"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func readerPodYaml(size int) io.Reader {
	r, w := io.Pipe()
	go func() {
		defer func() {
			_ = w.Close()
		}()
		for i := 0; i < size; i++ {
			_, _ = fmt.Fprintf(w, podYaml, i, i)
		}
	}()
	return r
}

var podYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: fake-pod-%06d
  namespace: default
  uid: 00000000-0000-0000-0001-%012d
spec:
  containers:
  - image: busybox
    name: container-0
  nodeName: fake-node-000000
---
`

func scaleCreatePodWithHack(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "hack", "put", "--path", "-")
	scaleCmd.Stdin = readerPodYaml(size)

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "Running", size, 5, 20, scaleCmd.Start); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func readerPodDeleteYaml(size int) io.Reader {
	r, w := io.Pipe()
	go func() {
		now := time.Now().UTC().Format(time.RFC3339)
		defer func() {
			_ = w.Close()
		}()
		for i := 0; i < size; i++ {
			_, _ = fmt.Fprintf(w, podDeleteYaml, i, i, now)
		}
	}()
	return r
}

var podDeleteYaml = `
apiVersion: v1
kind: Pod
metadata:
  name: fake-pod-%06d
  namespace: default
  uid: 00000000-0000-0000-0001-%012d
  deletionTimestamp: %s
spec:
  containers:
  - image: busybox
    name: container-0
  nodeName: fake-node-000000
---
`

func scaleDeletePodWithHack(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "hack", "put", "--path", "-")
	scaleCmd.Stdin = readerPodDeleteYaml(size)

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "Terminating", size, 0, 0, scaleCmd.Start); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func readerNodeYaml(size int) io.Reader {
	r, w := io.Pipe()
	go func() {
		defer func() {
			_ = w.Close()
		}()
		for i := 0; i < size; i++ {
			_, _ = fmt.Fprintf(w, nodeYaml, i, i)
		}
	}()
	return r
}

var nodeYaml = `
apiVersion: v1
kind: Node
metadata:
  annotations:
    kwok.x-k8s.io/node: fake
    node.alpha.kubernetes.io/ttl: "0"
  labels:
    beta.kubernetes.io/arch: amd64
    beta.kubernetes.io/os: linux
    kubernetes.io/arch: amd64
    kubernetes.io/os: linux
    kubernetes.io/role: agent
    node-role.kubernetes.io/agent: ""
    type: kwok
  name: fake-node-%06d
  uid: 00000000-1000-0000-0000-%012d
status:
  allocatable:
    cpu: "32"
    memory: 256Gi
    pods: "110"
  capacity:
    cpu: "32"
    memory: 256Gi
    pods: "110"
---
`

func scaleCreateNodeWithHack(ctx context.Context, t *testing.T, kwokctlPath string, name string, size int) error {
	scaleCmd := exec.CommandContext(ctx, kwokctlPath, "--name", name, "hack", "put", "--path", "-")
	scaleCmd.Stdin = readerNodeYaml(size)

	if err := waitResource(ctx, t, kwokctlPath, name, "Node", "Ready", size, 10, 20, scaleCmd.Start); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func CaseBenchmarkWithHack(kwokctlPath, clusterName string) *features.FeatureBuilder {
	return features.New("Benchmark").
		Assess("Create nodes", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			err := scaleCreateNodeWithHack(ctx0, t, kwokctlPath, clusterName, 5000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Create pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			err := scaleCreatePodWithHack(ctx0, t, kwokctlPath, clusterName, 10000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Delete pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			err := scaleDeletePodWithHack(ctx0, t, kwokctlPath, clusterName, 10000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
