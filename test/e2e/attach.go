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
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseAttach creates a feature that tests attach
func CaseAttach(kwokctlPath, clusterName, nodeName, namespace, tmpDir string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()

	return features.New("Pod Attach").
		Setup(helper.CreateNode(node)).
		Setup(helper.CreatePod(pod0)).
		Teardown(helper.DeletePod(pod0)).
		Teardown(helper.DeleteNode(node)).
		Assess("test attach", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("test attach")

			//nolint:gosec
			f, err := os.OpenFile(path.Join(tmpDir, "attach.log"), os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				err = f.Close()
				if err != nil {
					t.Fatal(err)
				}
				_ = os.Remove(path.Join(tmpDir, "attach.log"))
			}()

			buf := bytes.NewBuffer(nil)

			cmd, err := exec.Command(exec.WithWriteTo(exec.WithFork(ctx, true), buf), kwokctlPath, "--name", clusterName, "kubectl", "attach", "-n", namespace, "pod0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				err = cmd.Process.Kill()
				if err != nil {
					t.Fatal(err)
				}
			}()

			for i := 0; i != 30; i++ {
				_, _ = f.Write([]byte("2016-10-06T00:00:00Z stdout F wait\n"))
				_ = f.Sync()
				if buf.String() != "" {
					break
				}
				time.Sleep(1 * time.Second)
			}

			_, _ = f.Write([]byte("2016-10-06T00:00:00Z stdout F attach content 1\n"))
			_, _ = f.Write([]byte("2016-10-06T00:00:00Z stdout F attach content 2\n"))
			_, _ = f.Write([]byte("2016-10-06T00:00:00Z stdout F attach content 3\n"))

			want := "attach content 1\nattach content 2\nattach content 3\n"
			for i := 0; i != 30; i++ {
				got := strings.ReplaceAll(buf.String(), "wait\n", "")
				if got == want {
					break
				}
				time.Sleep(1 * time.Second)
			}
			got := strings.ReplaceAll(buf.String(), "wait\n", "")
			if got != want {
				t.Fatalf("want %s, got %s", want, got)
			}

			return ctx
		})
}
