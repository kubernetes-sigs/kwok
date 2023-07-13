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
	"math"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// HistogramOpts provides configuration options for Histogram.
type HistogramOpts struct {
	// Namespace, Subsystem, and Name are components of the fully-qualified
	// name of the Histogram (created by joining these components with
	// "_"). Only Name is mandatory, the others merely help structuring the
	// name. Note that the fully-qualified name of the Histogram must be a
	// valid Prometheus metric name.
	Namespace string
	Subsystem string
	Name      string

	// Help provides information about this Histogram.
	//
	// Metrics with the same fully-qualified name must have the same Help
	// string.
	Help string

	// ConstLabels are used to attach fixed labels to this metric. Metrics
	// with the same fully-qualified name must have the same label names in
	// their ConstLabels.
	//
	// ConstLabels are only used rarely. In particular, do not use them to
	// attach the same labels to all your metrics. Those use cases are
	// better covered by target labels set by the scraping Prometheus
	// server, or by one specific metric (e.g. a build_info or a
	// machine_role metric). See also
	// https://prometheus.io/docs/instrumenting/writing_exporters/#target-labels-not-static-scraped-labels
	ConstLabels prometheus.Labels
	Buckets     []float64
}

// histogram is custom type emulating prometheus.Histogram
type histogram struct {
	desc *prometheus.Desc

	// buckets are the upper bounds of the buckets
	buckets []float64

	// stored is a map of le -> count
	stored maps.SyncMap[float64, uint64]
}

// Histogram is a metric to track distributions of events.
type Histogram interface {
	prometheus.Metric
	prometheus.Collector
	Set(le float64, val uint64)
}

// NewHistogram creates new Histogram based on Histogram options
func NewHistogram(opts HistogramOpts) Histogram {
	desc := prometheus.NewDesc(
		prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		opts.Help,
		nil,
		opts.ConstLabels,
	)

	buckets := opts.Buckets
	sort.Float64s(buckets)

	his := &histogram{
		desc:    desc,
		buckets: buckets,
	}
	return his
}

// Desc returns prometheus.Desc used by every Prometheus Metric.
func (h *histogram) Desc() *prometheus.Desc {
	return h.desc
}

var inf = math.Inf(1)

// Write writes out histogram data to the Metric dto.
func (h *histogram) Write(out *dto.Metric) error {
	buckets := slices.Map(h.buckets, func(le float64) *dto.Bucket {
		return &dto.Bucket{
			CumulativeCount: format.Ptr[uint64](0),
			UpperBound:      format.Ptr(le),
		}
	})
	buckets = append(buckets, &dto.Bucket{
		CumulativeCount: format.Ptr[uint64](0),
		UpperBound:      format.Ptr(inf),
	})

	bucketsIndex := 0

	keys := h.stored.Keys()
	sort.Float64s(keys)

	var count uint64
	var sum float64
	for _, le := range keys {
		// cumulative count of previous buckets
		for bucketsIndex < len(buckets) && le > *buckets[bucketsIndex].UpperBound {
			bucketsIndex++
			buckets[bucketsIndex].CumulativeCount = format.Ptr(*buckets[bucketsIndex].CumulativeCount + count)
		}

		val, _ := h.stored.Load(le)
		// cumulative count of current bucket
		buckets[bucketsIndex].CumulativeCount = format.Ptr(*buckets[bucketsIndex].CumulativeCount + val)

		// cumulative count and sum
		count += val
		sum += le * float64(val)
	}

	his := &dto.Histogram{
		Bucket:      buckets,
		SampleCount: format.Ptr(count),
		SampleSum:   format.Ptr(sum),
	}

	out.Histogram = his

	return nil
}

// Describe sends metric description to a channel.
func (h *histogram) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.desc
}

// Collect sends histogram to a prometheus Metric channel.
func (h *histogram) Collect(ch chan<- prometheus.Metric) {
	ch <- h
}

// Set sets value for a given le.
func (h *histogram) Set(le float64, val uint64) {
	h.stored.Store(le, val)
}
