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

func TestAddWeightAfterWithPositiveDuration(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	pdq := NewWeightDelayingQueue[string](fakeClock)

	pdq.AddWeightAfter("foo", 1, 500*time.Millisecond)

	fakeClock.Step(100 * time.Millisecond)
	err := checkIncreased(pdq)
	if err != nil {
		t.Fatal(err)
	}

	fakeClock.Step(500 * time.Millisecond)
	err = checkLength(pdq, 1)
	if err != nil {
		t.Fatal(err)
	}

	pdq.AddWeightAfter("bar", 1, 500*time.Millisecond)

	fakeClock.Step(100 * time.Millisecond)
	err = checkIncreased(pdq)
	if err != nil {
		t.Fatal(err)
	}

	fakeClock.Step(500 * time.Millisecond)
	err = checkLength(pdq, 2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelWithExistingItem(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	pdq := NewWeightDelayingQueue[string](fakeClock)

	item := "foo"
	pdq.AddWeightAfter(item, 1, 500*time.Millisecond)

	canceled := pdq.Cancel(item)
	if !canceled {
		t.Fatal("expected true, got false")
	}

	err := checkIncreased(pdq)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancelWithNonExistingItem(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	pdq := NewWeightDelayingQueue[string](fakeClock)

	item := "foo"

	canceled := pdq.Cancel(item)
	if canceled {
		t.Fatal("expected false, got true")
	}
}
