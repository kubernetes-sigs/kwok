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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// prometheusTargetsResponse represents the Prometheus /api/v1/targets response.
type prometheusTargetsResponse struct {
	Data struct {
		ActiveTargets []prometheusTarget `json:"activeTargets"`
	} `json:"data"`
}

// prometheusTarget represents a single target in the Prometheus targets response.
type prometheusTarget struct {
	Labels map[string]string `json:"labels"`
	Health string            `json:"health"`
}

// CaseMetrics defines a feature test suite for verifying that all
// component metrics endpoints are successfully scraped by Prometheus.
// It queries the Prometheus API and checks each expected component's target
// health individually, rather than just counting the total number of healthy targets.
func CaseMetrics(expectedMetricsComponents ...string) *features.FeatureBuilder {
	return features.New("Metrics").
		Assess("All component metrics endpoints are healthy", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			if err := testPrometheusMetrics(expectedMetricsComponents); err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}

// testPrometheusMetrics queries the Prometheus targets API and verifies that
// every expected component's metrics endpoint is being scraped successfully.
func testPrometheusMetrics(expectedMetricsComponents []string) error {
	var lastErr error

	err := wait.For(
		func(ctx context.Context) (done bool, err error) {
			if err := checkPrometheusTargets(expectedMetricsComponents); err != nil {
				lastErr = err
				return false, nil
			}
			return true, nil
		},
		wait.WithTimeout(2*time.Minute),
		wait.WithInterval(5*time.Second),
	)
	if err != nil {
		if lastErr != nil {
			return fmt.Errorf("prometheus metrics check failed after retries: %w", lastErr)
		}
		return fmt.Errorf("prometheus metrics check timed out: %w", err)
	}
	return nil
}

// checkPrometheusTargets queries the Prometheus API and validates all expected
// component targets are present and healthy.
func checkPrometheusTargets(expectedMetricsComponents []string) error {
	resp, err := http.Get("http://127.0.0.1:9090/api/v1/targets")
	if err != nil {
		return fmt.Errorf("failed to query prometheus targets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus targets API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read prometheus response: %w", err)
	}

	var result prometheusTargetsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse prometheus targets response: %w", err)
	}

	// Build a map of job -> health from active targets
	targetHealth := make(map[string]string)
	for _, t := range result.Data.ActiveTargets {
		if job, ok := t.Labels["job"]; ok {
			targetHealth[job] = t.Health
		}
	}

	// Check each expected component
	var unhealthy []string
	var missing []string
	for _, job := range expectedMetricsComponents {
		health, ok := targetHealth[job]
		if !ok {
			missing = append(missing, job)
		} else if health != "up" {
			unhealthy = append(unhealthy, job)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing prometheus targets for components: [%s]",
			strings.Join(missing, ", "))
	}
	if len(unhealthy) > 0 {
		return fmt.Errorf("unhealthy prometheus targets for components: [%s]",
			strings.Join(unhealthy, ", "))
	}
	return nil
}
