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

package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewGauge(t *testing.T) {
	// Define gauge options
	opts := GaugeOpts{
		Name: "test_gauge",
		Help: "This is a test gauge",
	}

	// Create a new gauge
	gauge := NewGauge(opts)

	// Ensure the gauge starts at 0
	if err := testutil.CollectAndCompare(gauge, strings.NewReader(`
		# HELP test_gauge This is a test gauge
		# TYPE test_gauge gauge
		test_gauge 0
	`)); err != nil {
		t.Fatalf("unexpected metrics for starting value: %s", err)
	}

	// Increment the gauge
	gauge.Inc()
	if err := testutil.CollectAndCompare(gauge, strings.NewReader(`
		# HELP test_gauge This is a test gauge
		# TYPE test_gauge gauge
		test_gauge 1
	`)); err != nil {
		t.Fatalf("unexpected metrics after increment: %s", err)
	}

	// Decrement the gauge
	gauge.Dec()
	if err := testutil.CollectAndCompare(gauge, strings.NewReader(`
		# HELP test_gauge This is a test gauge
		# TYPE test_gauge gauge
		test_gauge 0
	`)); err != nil {
		t.Fatalf("unexpected metrics after decrement: %s", err)
	}

	// Set the gauge to a specific value
	gauge.Set(42)
	if err := testutil.CollectAndCompare(gauge, strings.NewReader(`
		# HELP test_gauge This is a test gauge
		# TYPE test_gauge gauge
		test_gauge 42
	`)); err != nil {
		t.Fatalf("unexpected metrics after set: %s", err)
	}
}
