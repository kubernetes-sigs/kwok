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

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

func CaseHack(kwokctlPath, clusterName, nodeName string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		Build()
	return features.New("Hack Data").
		Setup(helper.CreateNode(node)).
		Teardown(helper.DeleteNode(node)).
		Assess("Hack Data", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			buf0 := &bytes.Buffer{}
			_, err := exec.Command(exec.WithWriteTo(ctx, buf0), kwokctlPath, "--name", clusterName, "hack", "get", "no", nodeName)
			if err != nil {
				t.Fatal(err)
			}
			if buf0.Len() == 0 {
				t.Fatalf("failed hack get svc")
			}

			buf1 := &bytes.Buffer{}
			_, err = exec.Command(exec.WithWriteTo(ctx, buf1), kwokctlPath, "--name", clusterName, "hack", "delete", "no", nodeName)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(buf1.String()) != "/registry/minions/"+nodeName {
				t.Fatalf("failed hack delete node %q", buf1.String())
			}

			_, err = exec.Command(exec.WithReadFrom(ctx, bytes.NewBuffer(buf0.Bytes())), kwokctlPath, "--name", clusterName, "hack", "put", "no", nodeName, "--path", "-")
			if err != nil {
				t.Fatal(err)
			}

			buf2 := &bytes.Buffer{}
			_, err = exec.Command(exec.WithWriteTo(ctx, buf2), kwokctlPath, "--name", clusterName, "hack", "get", "no", nodeName)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(buf0.String(), buf2.String()); diff != "" {
				t.Errorf("failed hack put svc: %s", diff)
			}
			return ctx
		})
}
