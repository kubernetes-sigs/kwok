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
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseRecording is the recording of a test case.
func CaseRecording(kwokctlPath, clusterName string, tmpDir string) *features.FeatureBuilder {
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

	recordingPath := path.Join(tmpDir, "recording.yaml")

	var tmpCmd *exec.Cmd
	return features.New("Recording").
		Setup(helper.CreateNode(node0)).
		Setup(helper.CreatePod(pod0)).
		Assess("test record", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			cmd, err := exec.Command(exec.WithFork(ctx, true), kwokctlPath, "--name", clusterName, "kectl", "snapshot", "record", "--path", recordingPath)
			if err != nil {
				t.Fatal(err)
			}
			tmpCmd = cmd

			time.Sleep(1 * time.Second)
			return ctx
		}).
		Assess("create node1", helper.CreateNode(node1)).
		Assess("create pod1", helper.CreatePod(pod1)).
		Assess("check done", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctx, err := helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err = helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			var pods corev1.PodList
			err = client.List(ctx, &pods)
			if err != nil {
				t.Fatal(err)
			}

			podItems := slices.Filter(pods.Items, func(pod corev1.Pod) bool {
				return strings.HasPrefix(pod.Name, "pod")
			})
			if len(podItems) != 2 {
				t.Fatalf("pods not ready: %v", podItems)
			}

			var nodes corev1.NodeList
			err = client.List(ctx, &nodes)
			if err != nil {
				t.Fatal(err)
			}

			nodeItems := slices.Filter(nodes.Items, func(node corev1.Node) bool {
				return strings.HasPrefix(node.Name, "node")
			})
			if len(nodeItems) != 2 {
				t.Fatalf("nodes not ready: %v", nodeItems)
			}

			return ctx
		}).
		Assess("finish record", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			time.Sleep(1 * time.Second)

			sig := os.Interrupt
			if runtime.GOOS == "winddows" {
				sig = os.Kill
			}
			err := tmpCmd.Process.Signal(sig)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("delete pod1", helper.DeletePod(pod1)).
		Assess("delete node0", helper.DeleteNode(node0)).
		Assess("delete node1", helper.DeleteNode(node1)).
		Assess("test replay", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "--name", clusterName, "kectl", "snapshot", "replay", "--path", recordingPath)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("check done", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctx, err := helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err = helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			var pods corev1.PodList
			err = client.List(ctx, &pods)
			if err != nil {
				t.Fatal(err)
			}

			podItems := slices.Filter(pods.Items, func(pod corev1.Pod) bool {
				return strings.HasPrefix(pod.Name, "pod")
			})
			if len(podItems) != 2 {
				t.Fatalf("pods not ready: %v", podItems)
			}

			var nodes corev1.NodeList
			err = client.List(ctx, &nodes)
			if err != nil {
				t.Fatal(err)
			}

			nodeItems := slices.Filter(nodes.Items, func(node corev1.Node) bool {
				return strings.HasPrefix(node.Name, "node")
			})
			if len(nodeItems) != 2 {
				t.Fatalf("nodes not ready: %v", nodeItems)
			}

			return ctx
		}).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("delete pod1", helper.DeletePod(pod1)).
		Assess("delete node0", helper.DeleteNode(node0)).
		Assess("delete node1", helper.DeleteNode(node1))
}

// CaseRecordingExternal is the recording of a test case.
func CaseRecordingExternal(kwokctlPath, clusterName string, tmpDir string) *features.FeatureBuilder {
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

	recordingPath := path.Join(tmpDir, "recording.yaml")

	homeDir, _ := os.UserHomeDir()
	kubeconfigPath := filepath.Join(homeDir, ".kube/config")

	var tmpCmd *exec.Cmd
	return features.New("Recording").
		Setup(helper.CreateNode(node0)).
		Setup(helper.CreatePod(pod0)).
		Assess("test record", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			cmd, err := exec.Command(exec.WithFork(ctx, true), kwokctlPath, "snapshot", "export", "--kubeconfig", kubeconfigPath, "--record", "--name", clusterName, "--path", recordingPath)
			if err != nil {
				t.Fatal(err)
			}
			tmpCmd = cmd

			time.Sleep(1 * time.Second)
			return ctx
		}).
		Assess("create node1", helper.CreateNode(node1)).
		Assess("create pod1", helper.CreatePod(pod1)).
		Assess("check done", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctx, err := helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err = helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			var pods corev1.PodList
			err = client.List(ctx, &pods)
			if err != nil {
				t.Fatal(err)
			}

			podItems := slices.Filter(pods.Items, func(pod corev1.Pod) bool {
				return strings.HasPrefix(pod.Name, "pod")
			})
			if len(podItems) != 2 {
				t.Fatalf("pods not ready: %v", podItems)
			}

			var nodes corev1.NodeList
			err = client.List(ctx, &nodes)
			if err != nil {
				t.Fatal(err)
			}

			nodeItems := slices.Filter(nodes.Items, func(node corev1.Node) bool {
				return strings.HasPrefix(node.Name, "node")
			})
			if len(nodeItems) != 2 {
				t.Fatalf("nodes not ready: %v", nodeItems)
			}

			return ctx
		}).
		Assess("finish record", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			time.Sleep(1 * time.Second)

			sig := os.Interrupt
			if runtime.GOOS == "winddows" {
				sig = os.Kill
			}
			err := tmpCmd.Process.Signal(sig)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("delete pod1", helper.DeletePod(pod1)).
		Assess("delete node0", helper.DeleteNode(node0)).
		Assess("delete node1", helper.DeleteNode(node1)).
		Assess("test replay", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "--name", clusterName, "kectl", "snapshot", "replay", "--path", recordingPath)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("check done", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctx, err := helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err = helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			var pods corev1.PodList
			err = client.List(ctx, &pods)
			if err != nil {
				t.Fatal(err)
			}

			podItems := slices.Filter(pods.Items, func(pod corev1.Pod) bool {
				return strings.HasPrefix(pod.Name, "pod")
			})
			if len(podItems) != 2 {
				t.Fatalf("pods not ready: %v", podItems)
			}

			var nodes corev1.NodeList
			err = client.List(ctx, &nodes)
			if err != nil {
				t.Fatal(err)
			}

			nodeItems := slices.Filter(nodes.Items, func(node corev1.Node) bool {
				return strings.HasPrefix(node.Name, "node")
			})
			if len(nodeItems) != 2 {
				t.Fatalf("nodes not ready: %v", nodeItems)
			}

			return ctx
		}).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("delete pod1", helper.DeletePod(pod1)).
		Assess("delete node0", helper.DeleteNode(node0)).
		Assess("delete node1", helper.DeleteNode(node1))
}
