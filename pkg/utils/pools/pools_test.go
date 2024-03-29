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
