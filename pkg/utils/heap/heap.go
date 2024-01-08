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
	"cmp"
	"container/heap"
)

// Heap is a generic heap implementation.
type Heap[K cmp.Ordered, T comparable] struct {
	entries waitEntries[K, T]
	indexes map[T]*waitEntry[K, T]
}

// NewHeap creates a new heap.
func NewHeap[K cmp.Ordered, T comparable]() *Heap[K, T] {
	return &Heap[K, T]{
		indexes: map[T]*waitEntry[K, T]{},
	}
}

// Push adds an item to the queue.
func (h *Heap[K, T]) Push(key K, data T) {
	entry := &waitEntry[K, T]{key: key, data: data}
	heap.Push(&h.entries, entry)
	h.indexes[data] = entry
}

// Pop removes an item from the queue.
func (h *Heap[K, T]) Pop() (k K, v T, ok bool) {
	if len(h.entries) == 0 {
		return k, v, false
	}
	item := heap.Pop(&h.entries).(*waitEntry[K, T])
	delete(h.indexes, item.data)
	return item.key, item.data, true
}

// Peek returns the next item in the queue without removing it.
func (h *Heap[K, T]) Peek() (k K, v T, ok bool) {
	if len(h.entries) == 0 {
		return k, v, false
	}
	item := h.entries[0]
	return item.key, item.data, true
}

// Remove removes an item from the queue.
func (h *Heap[K, T]) Remove(data T) bool {
	if item, ok := h.indexes[data]; ok && item.index >= 0 {
		heap.Remove(&h.entries, item.index)
		delete(h.indexes, data)
		return true
	}
	return false
}

// Len returns the number of items in the queue.
func (h *Heap[K, T]) Len() int {
	return h.entries.Len()
}

// waitEntry is an entry in the delayingQueue heap.
type waitEntry[K cmp.Ordered, T any] struct {
	key   K
	data  T
	index int
}

type waitEntries[K cmp.Ordered, T any] []*waitEntry[K, T]

func (w waitEntries[K, T]) Len() int {
	return len(w)
}

func (w waitEntries[K, T]) Less(i, j int) bool {
	return cmp.Less[K](w[i].key, w[j].key)
}

func (w waitEntries[K, T]) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
	w[i].index = i
	w[j].index = j
}

// Push adds an item to the queue, should not be called directly instead use `heap.Push`.
func (w *waitEntries[K, T]) Push(x any) {
	n := len(*w)
	item := x.(*waitEntry[K, T])
	item.index = n
	*w = append(*w, item)
}

// Pop removes an item from the queue, should not be called directly instead use `heap.Pop`.
func (w *waitEntries[K, T]) Pop() any {
	n := len(*w)
	item := (*w)[n-1]
	item.index = -1
	*w = (*w)[0:(n - 1)]
	return item
}
