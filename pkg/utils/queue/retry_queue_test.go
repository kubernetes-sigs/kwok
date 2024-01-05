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

package queue

import (
	"testing"
	"time"

	fakeclock "k8s.io/utils/clock/testing"
)

func TestBackoffRetryQueueQueueSimple(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())

	// the test values are designed to hit the maxInterval in the third revision ideally
	// the first(initial) backoff period (s):   [10.0, 11.0]
	// the second backoff period (s):           [20.0, 24.2]
	// the third backoff period (s) (hit max):  30
	q := NewBackoffRetryQueue[string](fakeClock, 10*time.Second, 30*time.Second, 2, 0.1)

	item := "foo"
	q.Add(item)
	fakeClock.Step(9 * time.Second)
	err := checkIncreased[string](q)
	if err != nil {
		t.Fatal()
	}
	// total 11s passed until the item is first added
	fakeClock.Step(2 * time.Second)
	err = checkLength[string](q, 1)
	if err != nil {
		t.Fatal()
	}

	x, ok := q.Get()
	if !ok || x != item {
		t.Fatalf("expected %s, got %s", item, x)
	}

	q.Add(item)
	// total 19s passed until the item was re added
	fakeClock.Step(19 * time.Second)
	err = checkIncreased[string](q)
	if err != nil {
		t.Fatal(err)
	}

	// total 25s passed until the item was re added
	fakeClock.Step(6 * time.Second)
	err = checkLength[string](q, 1)
	if err != nil {
		t.Fatal()
	}
	x, ok = q.Get()
	if !ok || x != item {
		t.Fatalf("expected %s, got %s", item, x)
	}

	q.Add(item)
	// total 29s passed until the item was added the third time
	fakeClock.Step(29 * time.Second)
	err = checkIncreased[string](q)
	if err != nil {
		t.Fatal(err)
	}

	// total 31s passed until the item was added the third time
	fakeClock.Step(1 * time.Second)
	err = checkLength[string](q, 1)
	if err != nil {
		t.Fatal(err)
	}
	x, ok = q.Get()
	if !ok || x != item {
		t.Fatalf("expected %s, got %s", item, x)
	}

	// reset
	q.Remove(item)
	q.Add(item)

	fakeClock.Step(9 * time.Second)
	// total 9s passed until the item was reset
	err = checkIncreased[string](q)
	if err != nil {
		t.Fatal(err)
	}
	// total 11s passed until the item was reset
	fakeClock.Step(2 * time.Second)
	err = checkLength[string](q, 1)
	if err != nil {
		t.Fatal(err)
	}
	x, ok = q.Get()
	if !ok || x != item {
		t.Fatalf("expected %s, got %s", item, x)
	}
}
