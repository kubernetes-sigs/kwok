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
	"container/heap"
	"sync"
	"time"
)

// Clock is an interface that returns the current time and duration since a given time.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
	Sleep(d time.Duration)
}

// DelayingQueue is a generic queue interface that supports adding items after
type DelayingQueue[T comparable] interface {
	Queue[T]
	// AddAfter adds an item to the queue after the indicated duration has passed
	AddAfter(item T, duration time.Duration) bool
	// Cancel removes an item from the queue if it has not yet been processed
	Cancel(item T) bool
	// Pending returns the number of items in the queue that have not yet been processed
	Pending() int
}

// delayingQueue is a generic DelayingQueue implementation.
type delayingQueue[T comparable] struct {
	Queue[T]

	clock Clock

	mut sync.Mutex

	heap    waitEntries[T]
	entries map[T]*waitEntry[T]

	signal chan struct{}
}

// NewDelayingQueue returns a new DelayingQueue.
func NewDelayingQueue[T comparable](clock Clock) DelayingQueue[T] {
	q := &delayingQueue[T]{
		Queue:   NewQueue[T](),
		clock:   clock,
		heap:    waitEntries[T]{},
		entries: make(map[T]*waitEntry[T]),
		signal:  make(chan struct{}, 1),
	}
	go q.loopWorker()
	return q
}

func (q *delayingQueue[T]) Pending() int {
	q.mut.Lock()
	defer q.mut.Unlock()
	return len(q.heap)
}

func (q *delayingQueue[T]) AddAfter(item T, duration time.Duration) bool {
	if duration <= 0 {
		q.Queue.Add(item)
		return true
	}

	q.mut.Lock()
	defer q.mut.Unlock()

	_, ok := q.entries[item]
	if ok {
		return false
	}

	entry := &waitEntry[T]{data: item, readyAt: q.clock.Now().Add(duration)}
	heap.Push(&q.heap, entry)
	q.entries[item] = entry

	select {
	case q.signal <- struct{}{}:
	default:
	}
	return true
}

func (q *delayingQueue[T]) loopWorker() {
	for {
		t, ok, next := q.next()
		if ok {
			q.Queue.Add(t)
			continue
		}

		delay := 10 * time.Second
		if next != nil && *next < delay {
			delay = *next
		}
		select {
		case <-q.clock.After(delay):
		case <-q.signal:
		}
	}
}

func (q *delayingQueue[T]) next() (t T, ok bool, wait *time.Duration) {
	q.mut.Lock()
	defer q.mut.Unlock()

	if len(q.heap) == 0 {
		return t, false, nil
	}
	entry := q.heap[0]
	waitDuration := entry.readyAt.Sub(q.clock.Now())
	if waitDuration > 0 {
		return t, false, &waitDuration
	}

	entry = heap.Pop(&q.heap).(*waitEntry[T])
	delete(q.entries, entry.data)
	return entry.data, true, nil
}

func (q *delayingQueue[T]) Cancel(item T) bool {
	q.mut.Lock()
	defer q.mut.Unlock()
	return q.cancel(item)
}

func (q *delayingQueue[T]) cancel(item T) bool {
	entry, ok := q.entries[item]
	if !ok {
		return false
	}

	heap.Remove(&q.heap, entry.index)
	delete(q.entries, item)
	return true
}

// waitEntry is an entry in the delayingQueue heap.
type waitEntry[T any] struct {
	data    T
	readyAt time.Time
	// index in the heap
	index int
}

type waitEntries[T any] []*waitEntry[T]

func (w waitEntries[T]) Len() int {
	return len(w)
}

func (w waitEntries[T]) Less(i, j int) bool {
	return w[i].readyAt.Before(w[j].readyAt)
}

func (w waitEntries[T]) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
	w[i].index = i
	w[j].index = j
}

// Push adds an item to the queue, should not be called directly instead use `heap.Push`.
func (w *waitEntries[T]) Push(x any) {
	n := len(*w)
	item := x.(*waitEntry[T])
	item.index = n
	*w = append(*w, item)
}

// Pop removes an item from the queue, should not be called directly instead use `heap.Pop`.
func (w *waitEntries[T]) Pop() any {
	n := len(*w)
	item := (*w)[n-1]
	item.index = -1
	*w = (*w)[0:(n - 1)]
	return item
}
