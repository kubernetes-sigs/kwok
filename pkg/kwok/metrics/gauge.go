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
	"github.com/prometheus/client_golang/prometheus"
)

type (
	// Gauge is a prometheus gauge that can be incremented and decremented.
	Gauge = prometheus.Gauge
	// GaugeOpts is a prometheus gauge options.
	GaugeOpts = prometheus.GaugeOpts
)

// NewGauge returns a new gauge.
func NewGauge(opts GaugeOpts) Gauge {
	return prometheus.NewGauge(opts)
}
