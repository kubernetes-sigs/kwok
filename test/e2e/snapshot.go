/*
Copyright 2023 The Kubernetes Authors.

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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseSnapshot is the snapshot of a test case.
func CaseSnapshot(kwokctlPath, clusterName string, tmpDir string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder("node0").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNodeName(node.Name).
		Build()

	dbPath := path.Join(tmpDir, "snapshot.db")

	return features.New("Snapshot").
		Setup(helper.CreateNode(node)).
		Setup(helper.CreatePod(pod0)).
		Assess("test snapshot", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "snapshot", "save", "--name", clusterName, "--path", dbPath)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod", helper.DeletePod(pod0)).
		Assess("delete node", helper.DeleteNode(node)).
		Assess("test restore", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := exec.Command(ctx, kwokctlPath, "snapshot", "restore", "--name", clusterName, "--path", dbPath)
			if err != nil {
				t.Fatal(err)
			}

			_ = os.Remove(dbPath)
			return ctx
		}).
		Assess("check restore", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, err := helper.WaitForAllNodesReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			_, err = helper.WaitForAllPodsReady()(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			var pod corev1.Pod
			err = client.Get(ctx, pod0.Name, pod0.Namespace, &pod)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod0", helper.DeletePod(pod0)).
		Assess("delete node0", helper.DeleteNode(node))
}
