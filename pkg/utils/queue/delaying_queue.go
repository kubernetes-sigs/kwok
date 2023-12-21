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
	"sync"
	"time"

	"sigs.k8s.io/kwok/pkg/utils/heap"
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
}

// delayingQueue is a generic DelayingQueue implementation.
type delayingQueue[T comparable] struct {
	Queue[T]

	clock Clock

	mut sync.Mutex

	heap *heap.Heap[int64, T]

	signal chan struct{}
}

// NewDelayingQueue returns a new DelayingQueue.
func NewDelayingQueue[T comparable](clock Clock) DelayingQueue[T] {
	q := &delayingQueue[T]{
		Queue:  NewQueue[T](),
		clock:  clock,
		heap:   heap.NewHeap[int64, T](),
		signal: make(chan struct{}, 1),
	}
	go q.loopWorker()
	return q
}

func (q *delayingQueue[T]) AddAfter(item T, duration time.Duration) bool {
	if duration <= 0 {
		q.Queue.Add(item)
		return true
	}

	q.mut.Lock()
	defer q.mut.Unlock()

	k := q.clock.Now().Add(duration).UnixNano()

	q.heap.Push(k, item)

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

	k, v, ok := q.heap.Peek()
	if !ok {
		return t, false, nil
	}

	waitDuration := time.Unix(0, k).Sub(q.clock.Now())
	if waitDuration > 0 {
		return v, false, &waitDuration
	}

	q.heap.Remove(v)
	return v, true, nil
}

func (q *delayingQueue[T]) Cancel(item T) bool {
	q.mut.Lock()
	defer q.mut.Unlock()
	return q.heap.Remove(item)
}
