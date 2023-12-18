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

package queue

import (
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	q := NewQueue[string]()

	first := "foo"
	second := "bar"

	q.Add(first)
	q.Add(second)

	if q.Len() != 2 {
		t.Fatalf("expected: %d, got: %d", 2, q.Len())
	}

	x := q.GetOrWait()
	if x != first {
		t.Fatalf("expected: %s, got: %s", first, x)
	}
	if q.Len() != 1 {
		t.Fatalf("expected: %d, got: %d", 1, q.Len())
	}

	x = q.GetOrWait()
	if x != second {
		t.Fatalf("expected: %s, got: %s", second, x)
	}
	if q.Len() != 0 {
		t.Fatalf("expected: %d, got: %d", 0, q.Len())
	}
}

func TestBlock(t *testing.T) {
	q := NewQueue[string]()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = q.GetOrWait()
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		// expected
	case <-done:
		t.Fatal("should block when queue is empty")
	}

	q.Add("foo")

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatal("should not block when queue holds elements")
	case <-done:
		// expected
	}
}
