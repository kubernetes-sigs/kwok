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
	"fmt"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
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

// waitComponentsReady waits until kwokctl reports every component of the cluster as Ready.
// kwokctl --wait logs a timeout without failing, so its exit code alone does not prove readiness.
func waitComponentsReady(ctx context.Context, kwokctlPath, clusterName string, timeout time.Duration) error {
	var lastErr error
	err := wait.For(
		func(ctx context.Context) (bool, error) {
			lastErr = checkComponentsReady(ctx, kwokctlPath, clusterName)
			return lastErr == nil, nil
		},
		wait.WithContext(ctx),
		wait.WithTimeout(timeout),
		wait.WithInterval(time.Second),
	)
	if err != nil && lastErr != nil {
		return lastErr
	}
	return err
}

func checkComponentsReady(ctx context.Context, kwokctlPath, clusterName string) error {
	buf := &bytes.Buffer{}
	_, err := utilsexec.Command(utilsexec.WithAllWriteTo(ctx, buf), kwokctlPath, "get", "components", "--name", clusterName, "--output", "wide")
	if err != nil {
		return fmt.Errorf("failed to get components: %w: %s", err, buf.String())
	}

	output := strings.TrimSpace(buf.String())
	var count int
	var notReady []string
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] == "NAME" {
			continue
		}
		count++
		if fields[1] != "Ready" {
			notReady = append(notReady, fields[0]+"="+fields[1])
		}
	}
	if count == 0 {
		return fmt.Errorf("no components found in output %q", output)
	}
	if len(notReady) != 0 {
		return fmt.Errorf("components are not ready: %v", notReady)
	}
	return nil
}
