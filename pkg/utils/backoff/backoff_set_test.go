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

package backoff

import (
	"testing"
	"time"
)

func TestBackoffSetSimple(t *testing.T) {
	// the test values are designed to hit the maxInterval in the third revision ideally
	cases := []struct {
		name         string
		jitter       float64
		factor       float64
		initInterval time.Duration
		maxInterval  time.Duration
	}{
		{
			// the first(initial) backoff period (s):   [10.0, 11.0]
			// the second backoff period (s):           [20.0, 24.2]
			// the third backoff period (s) (hit max):  30.0
			name:         "withJitter",
			jitter:       0.1,
			initInterval: 10 * time.Second,
			maxInterval:  30 * time.Second,
			factor:       2,
		},
		{
			// the first(initial) backoff period (s):   10.0
			// the second backoff period (s):           20.0
			// the third backoff period (s) (hit max):  30.0
			name:         "withoutJitter",
			jitter:       0,
			initInterval: 10 * time.Second,
			maxInterval:  30 * time.Second,
			factor:       2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			initInterval := 10 * time.Second
			maxInterval := 30 * time.Second
			item := "foo"

			bs := NewBackoffSet[string](
				WithMaxInterval(maxInterval),
				WithInitialInterval(initInterval),
				WithJitter(tc.jitter),
				WithFactor(tc.factor))

			firstDelay := bs.AddOrUpdate(item)
			// when jitter is zero, the following if statement is equivalent to `if firstDelay != initInterval `
			if float64(firstDelay) > (1+tc.jitter)*float64(initInterval) || firstDelay < initInterval {
				t.Fatal("the delay time is unexpected")
			}

			secondDelay := bs.AddOrUpdate(item)
			// when jitter is zero, the following if statement is equivalent to `if secondDelay != initInterval*DefaultFactor`
			if float64(secondDelay) > (1+tc.jitter)*float64(firstDelay)*tc.factor || float64(secondDelay) < float64(firstDelay)*tc.factor {
				t.Fatal("the delay time is unexpected")
			}

			// Should be capped at the maxInterval here
			thirdDelay := bs.AddOrUpdate(item)
			if thirdDelay != maxInterval {
				t.Fatal("the delay time is unexpected")
			}

			// Reset
			bs.Remove(item)
			delayAfterReset := bs.AddOrUpdate(item)
			if float64(delayAfterReset) > (1+tc.jitter)*float64(initInterval) || firstDelay < initInterval {
				t.Fatal("the delay time is unexpected")
			}
		})
	}
}
