/*
Copyright 2026 The Kubernetes Authors.

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

	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
)

// CaseGetClustersStatus checks if the cluster status is reported correctly.
func CaseGetClustersStatus(kwokctlPath, clusterName string) *features.FeatureBuilder {
	return features.New("Get Cluster Status").
		Assess("test get clusters status", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			buf := &bytes.Buffer{}
			_, err := utilsexec.Command(utilsexec.WithAllWriteTo(ctx, buf), kwokctlPath, "get", "clusters", "--output", "wide")
			if err != nil {
				t.Fatal(err)
			}

			output := strings.TrimSpace(buf.String())
			for line := range strings.SplitSeq(output, "\n") {
				fields := strings.Fields(line)
				if len(fields) < 3 || fields[0] != clusterName {
					continue
				}

				if fields[2] != "Ready" {
					t.Fatalf("cluster %q status should be Ready, got %q from output %q", clusterName, fields[2], output)
				}
				return ctx
			}

			t.Fatalf("cluster %q not found in output %q", clusterName, output)
			return ctx
		})
}
