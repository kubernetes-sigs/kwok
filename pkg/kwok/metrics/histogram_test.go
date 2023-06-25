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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	dto "github.com/prometheus/client_model/go"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestHistogramObserve(t *testing.T) {
	opts := HistogramOpts{
		Name:    "name",
		Help:    "help",
		Buckets: []float64{0.5, 1, 2.5, 5, 10},
	}

	data := map[float64]uint64{
		0.9:  0b_00_00_01,
		1.0:  0b_00_00_10,
		1.1:  0b_00_01_00,
		2.4:  0b_00_10_00,
		10.0: 0b_01_00_00,
		11.0: 0b_10_00_00,
	}

	his := NewHistogram(opts)
	for le, count := range data {
		his.Set(le, count)
	}

	var out dto.Metric
	if err := his.Write(&out); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	sampleCount := uint64(0)
	sampleSum := float64(0)
	for le, count := range data {
		sampleCount += count
		sampleSum += le * float64(count)
	}

	want := &dto.Histogram{
		SampleCount: format.Ptr(sampleCount),
		SampleSum:   format.Ptr(sampleSum),
		Bucket: []*dto.Bucket{
			{
				CumulativeCount: format.Ptr[uint64](0b_00_00_00),
				UpperBound:      format.Ptr(0.5),
			},
			{
				CumulativeCount: format.Ptr[uint64](0b_00_00_11),
				UpperBound:      format.Ptr(1.0),
			},
			{
				CumulativeCount: format.Ptr[uint64](0b_00_11_11),
				UpperBound:      format.Ptr(2.5),
			},
			{
				CumulativeCount: format.Ptr[uint64](0b_00_11_11),
				UpperBound:      format.Ptr(5.0),
			},
			{
				CumulativeCount: format.Ptr[uint64](0b_01_11_11),
				UpperBound:      format.Ptr(10.0),
			},
			{
				CumulativeCount: format.Ptr[uint64](0b_11_11_11),
				UpperBound:      format.Ptr(inf),
			},
		},
	}

	if diff := cmp.Diff(out.Histogram, want, cmpopts.IgnoreUnexported(dto.Histogram{}, dto.Bucket{})); diff != "" {
		t.Errorf("Histogram mismatch (-want +got):\n%s", diff)
	}
}
