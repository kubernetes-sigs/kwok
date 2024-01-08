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

package heap

import (
	"testing"
)

func TestNewHeap(t *testing.T) {
	h := NewHeap[int, string]()
	if h.Len() != 0 {
		t.Errorf("NewHeap() = %d; want 0", h.Len())
	}
}

func TestPushAndPop(t *testing.T) {
	h := NewHeap[int, string]()
	h.Push(1, "one")
	h.Push(2, "two")
	h.Push(3, "three")

	key, data, _ := h.Pop()
	if key != 1 || data != "one" {
		t.Errorf("Pop() = %d, %s; want 1, one", key, data)
	}

	key, data, _ = h.Pop()
	if key != 2 || data != "two" {
		t.Errorf("Pop() = %d, %s; want 2, two", key, data)
	}

	key, data, _ = h.Pop()
	if key != 3 || data != "three" {
		t.Errorf("Pop() = %d, %s; want 3, three", key, data)
	}
}

func TestPeek(t *testing.T) {
	h := NewHeap[int, string]()
	h.Push(1, "one")
	h.Push(2, "two")

	key, data, _ := h.Peek()
	if key != 1 || data != "one" {
		t.Errorf("Peek() = %d, %s; want 1, one", key, data)
	}
}

func TestLen(t *testing.T) {
	h := NewHeap[int, string]()
	if h.Len() != 0 {
		t.Errorf("Len() = %d; want 0", h.Len())
	}

	h.Push(1, "one")
	if h.Len() != 1 {
		t.Errorf("Len() = %d; want 1", h.Len())
	}

	h.Push(2, "two")
	if h.Len() != 2 {
		t.Errorf("Len() = %d; want 2", h.Len())
	}

	h.Pop()
	if h.Len() != 1 {
		t.Errorf("Len() = %d; want 1", h.Len())
	}

	h.Pop()
	if h.Len() != 0 {
		t.Errorf("Len() = %d; want 0", h.Len())
	}
}

func TestPopEmptyHeap(t *testing.T) {
	h := NewHeap[int, string]()
	_, _, ok := h.Peek()
	if ok {
		t.Errorf("Peek() = %t; want false", ok)
	}
	_, _, ok = h.Pop()
	if ok {
		t.Errorf("Pop() = %t; want false", ok)
	}
	if h.Len() != 0 {
		t.Errorf("Len() = %d; want 0", h.Len())
	}
}

func TestRemoveExistingItem(t *testing.T) {
	h := NewHeap[int, string]()
	h.Push(1, "one")
	h.Push(2, "two")

	removed := h.Remove("one")
	if !removed {
		t.Errorf("Remove() = %t; want true", removed)
	}

	if h.Len() != 1 {
		t.Errorf("Len() = %d; want 1", h.Len())
	}
}

func TestRemoveNonExistingItem(t *testing.T) {
	h := NewHeap[int, string]()
	h.Push(1, "one")

	removed := h.Remove("two")
	if removed {
		t.Errorf("Remove() = %t; want false", removed)
	}

	if h.Len() != 1 {
		t.Errorf("Len() = %d; want 1", h.Len())
	}
}

func TestRemoveFromEmptyHeap(t *testing.T) {
	h := NewHeap[int, string]()

	removed := h.Remove("one")
	if removed {
		t.Errorf("Remove() = %t; want false", removed)
	}

	if h.Len() != 0 {
		t.Errorf("Len() = %d; want 0", h.Len())
	}
}
