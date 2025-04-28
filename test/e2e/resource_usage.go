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
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseResourceUsage defines a feature test suite for verifying resource usage metrics in a KWOK cluster.
// It creates nodes and pods, waits for them to be ready, and then tests the resource usage metrics
// using kubectl top command. The test ensures that the metrics API is available and returns
// expected resource usage values for the created pods.
func CaseResourceUsage(kwokctlPath, clusterName string) *features.FeatureBuilder {
	node0 := helper.NewNodeBuilder("node0").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNodeName(node0.Name).
		Build()

	node1 := helper.NewNodeBuilder("node1").
		Build()
	pod1 := helper.NewPodBuilder("pod1").
		WithNodeName(node1.Name).
		Build()

	return features.New("ResourceUsage").
		Setup(helper.CreateNode(node0)).
		Setup(helper.CreatePod(pod0)).
		Setup(helper.CreateNode(node1)).
		Setup(helper.CreatePod(pod1)).
		Assess("wait ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			_, err = helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			return ctx
		}).
		Assess("test usage", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			logger := log.FromContext(ctx)

			err := wait.For(
				func(ctx context.Context) (done bool, err error) {
					out := bytes.NewBuffer(nil)
					_, err = exec.Command(exec.WithAllWriteTo(ctx, out), kwokctlPath, "--name", clusterName, "kubectl", "top", "pod")
					if err != nil {
						logger.Error("kubectl top pod", err)
						return false, nil
					}

					output := out.String()
					if strings.Contains(output, "Metrics API not available") || strings.Contains(output, "metrics not available yet") {
						logger.Warn("kubectl top pod", "output", output)
						return false, nil
					}
					if !strings.Contains(output, "1Mi") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}
					if !strings.Contains(output, "1m") && !strings.Contains(output, "2m") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}
					return true, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(5*time.Minute),
			)
			if err != nil {
				t.Fatal(err)
				return ctx
			}

			return ctx
		}).
		Assess("test modify usage", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			logger := log.FromContext(ctx)

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Error(err)
			}

			err = client.Patch(ctx, pod0, k8s.Patch{
				PatchType: types.JSONPatchType,
				Data:      []byte(`[{"op": "add", "path": "/metadata/annotations", "value": {"kwok.x-k8s.io/usage-cpu": "100m", "kwok.x-k8s.io/usage-memory": "100Mi"}}]`),
			})
			if err != nil {
				t.Error(err)
			}

			err = wait.For(
				func(ctx context.Context) (done bool, err error) {
					out := bytes.NewBuffer(nil)
					_, err = exec.Command(exec.WithAllWriteTo(ctx, out), kwokctlPath, "--name", clusterName, "kubectl", "top", "pod")
					if err != nil {
						logger.Error("kubectl top pod", err)
						return false, nil
					}

					output := out.String()
					if strings.Contains(output, "Metrics API not available") || strings.Contains(output, "metrics not available yet") {
						logger.Warn("kubectl top pod", "output", output)
						return false, nil
					}
					if !strings.Contains(output, "1Mi") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}
					if !strings.Contains(output, "1m") && !strings.Contains(output, "2m") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}

					if !strings.Contains(output, "100Mi") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}
					if !strings.Contains(output, "100m") && !strings.Contains(output, "101m") {
						logger.Info("kubectl top pod", "output", output)
						return false, nil
					}
					return true, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(5*time.Minute),
			)
			if err != nil {
				t.Fatal(err)
				return ctx
			}

			return ctx
		})
}
