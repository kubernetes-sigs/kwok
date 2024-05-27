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

package metrics

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewCounter(t *testing.T) {
	// Define counter options
	opts := CounterOpts{
		Name: "test_counter",
		Help: "This is a test counter",
	}

	// Initialize a float64 value
	initialValue := float64(0)

	// Create a new counter and ensure its atomic pointer is initialized properly
	c := &counter{
		value: &atomic.Pointer[float64]{},
	}
	c.value.Store(&initialValue)
	c.CounterFunc = prometheus.NewCounterFunc(opts, func() float64 {
		return *c.value.Load()
	})

	// Register the counter with Prometheus's default registry
	prometheus.MustRegister(c)

	// Ensure the counter starts at 0
	if err := testutil.CollectAndCompare(c, strings.NewReader(`
		# HELP test_counter This is a test counter
		# TYPE test_counter counter
		test_counter 0
	`)); err != nil {
		t.Fatalf("unexpected metrics for starting value: %s", err)
	}

	// Set the counter to a specific value
	newValue := float64(42)
	c.Set(newValue)
	if err := testutil.CollectAndCompare(c, strings.NewReader(`
		# HELP test_counter This is a test counter
		# TYPE test_counter counter
		test_counter 42
	`)); err != nil {
		t.Fatalf("unexpected metrics after set: %s", err)
	}

	// Set the counter to another value
	newValue = float64(84)
	c.Set(newValue)
	if err := testutil.CollectAndCompare(c, strings.NewReader(`
		# HELP test_counter This is a test counter
		# TYPE test_counter counter
		test_counter 84
	`)); err != nil {
		t.Fatalf("unexpected metrics after second set: %s", err)
	}

	// Clean up the counter from the default registry
	prometheus.Unregister(c)
}
