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

package server

import (
	"testing"
	"time"
)

func TestUpdateCumulativeRemainsMonotonicWhenReplicaCountChanges(t *testing.T) {
	now := time.Unix(100, 0)
	cumulative := updateCumulative(cumulative{}, now, 500)
	steps := []struct {
		replicas int
		expected float64
	}{
		{replicas: 2, expected: 500},
		{replicas: 4, expected: 750},
		{replicas: 1, expected: 1750},
	}
	last := cumulative.value
	for _, step := range steps {
		now = now.Add(time.Second)
		cumulative = updateCumulative(cumulative, now, 1000/float64(step.replicas))
		if cumulative.value != step.expected {
			t.Fatalf("replicas %d: expected %v, got %v", step.replicas, step.expected, cumulative.value)
		}
		if cumulative.value < last {
			t.Fatalf("replicas %d: cumulative value decreased from %v to %v", step.replicas, last, cumulative.value)
		}
		last = cumulative.value
	}
}
