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

package pools

import (
	"sync/atomic"
	"testing"
)

func TestPool(t *testing.T) {
	index := 0
	pool := NewPool(func() int {
		index++
		return index
	})

	pool.Put(3)
	if pool.Get() != 3 {
		t.Errorf("expected 3, got %d", pool.Get())
	}

	if pool.Get() != 1 {
		t.Errorf("expected 1, got %d", pool.Get())
	}
	if pool.Get() != 2 {
		t.Errorf("expected 2, got %d", pool.Get())
	}

	pool.Put(1)
	if pool.Get() != 1 {
		t.Errorf("expected 1, got %d", pool.Get())
	}

	pool.Put(2)
	if pool.Get() != 2 {
		t.Errorf("expected 2, got %d", pool.Get())
	}

	if pool.Get() != 3 {
		t.Errorf("expected 3, got %d", pool.Get())
	}
}

func TestPool_ConcurrentAccess(t *testing.T) {
	var index int32
	pool := NewPool(func() int {
		return int(atomic.AddInt32(&index, 1))
	})

	// Test concurrent access to the pool
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(val int) {
			pool.Put(val)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	values := make(map[int]bool)
	for i := 0; i < 10; i++ {
		val := pool.Get()
		if val < 0 || val >= 10 {
			t.Errorf("unexpected value %d", val)
		}
		if values[val] {
			t.Errorf("duplicate value found: %d", val)
		}
		values[val] = true
	}
}
