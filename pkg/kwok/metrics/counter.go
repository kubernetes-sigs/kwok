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
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type (
	// CounterOpts is a prometheus counter options.
	CounterOpts = prometheus.CounterOpts
)

// counter is a prometheus counter that can be incremented and decremented.
type counter struct {
	value *atomic.Pointer[float64]
	prometheus.CounterFunc
}

// Counter is a prometheus counter that can be incremented and decremented.
type Counter interface {
	prometheus.Metric
	prometheus.Collector
	// Set sets the counter to the given value.
	Set(value float64)
}

// NewCounter returns a new counter.
func NewCounter(opts CounterOpts) Counter {
	c := &counter{
		value: &atomic.Pointer[float64]{},
	}
	c.value.Store(new(float64))
	c.CounterFunc = prometheus.NewCounterFunc(opts,
		func() float64 {
			return *c.value.Load()
		},
	)
	return c
}

// Set sets the counter to the given value.
func (c *counter) Set(value float64) {
	c.value.Store(&value)
}
