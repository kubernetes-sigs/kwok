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
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseExec creates a feature that tests exec
func CaseExec(kwokctlPath, clusterName, nodeName, namespace string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		Build()

	return features.New("Pod Exec").
		Setup(helper.CreateNode(node)).
		Setup(helper.CreatePod(pod0)).
		Teardown(helper.DeletePod(pod0)).
		Teardown(helper.DeleteNode(node)).
		Assess("test exec", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("test exec")
			buf := &bytes.Buffer{}
			_, err := exec.Command(exec.WithAllWriteTo(ctx, buf), kwokctlPath, "--name", clusterName, "kubectl", "exec", "-n", namespace, "pod0", "--", "env")
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(buf.String(), "TEST=test") {
				t.Fatalf("failed output %q", buf.String())
			}
			return ctx
		})
}
