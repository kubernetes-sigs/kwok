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
	"sort"
	"sync"
	"time"

	"sigs.k8s.io/kwok/pkg/utils/heap"
	"sigs.k8s.io/kwok/pkg/utils/maps"
)

// WeightDelayingQueue is a generic weight delaying queue interface.
type WeightDelayingQueue[T comparable] interface {
	Queue[T]
	delayingQueueInterface[T]
	weightQueueInterface[T]
	weightDelayingQueueInterface[T]
}

type weightDelayingQueueInterface[T comparable] interface {
	// AddWeightAfter adds an item to the queue with the given weight after the indicated duration has passed
	// if weight is zero, it is the highest weight.
	// otherwise, the weight ranges from 1 to n, with higher numbers meaning higher weight.
	AddWeightAfter(item T, weight int, duration time.Duration)
}

type weightDelayingQueue[T comparable] struct {
	WeightQueue[T]
	orders []int

	clock Clock

	heap  *heap.Heap[int64, T]
	heaps map[int]*heap.Heap[int64, T]

	signal chan struct{}
	mut    sync.Mutex
}

// NewWeightDelayingQueue returns a new WeightDelayingQueue.
func NewWeightDelayingQueue[T comparable](clock Clock) WeightDelayingQueue[T] {
	q := &weightDelayingQueue[T]{
		WeightQueue: NewWeightQueue[T](),
		clock:       clock,
		heap:        heap.NewHeap[int64, T](),
		heaps:       map[int]*heap.Heap[int64, T]{},
		signal:      make(chan struct{}, 1),
	}
	go q.loopWorker()
	return q
}

func (q *weightDelayingQueue[T]) AddAfter(item T, duration time.Duration) {
	q.AddWeightAfter(item, 0, duration)
}

func (q *weightDelayingQueue[T]) AddWeightAfter(item T, weight int, duration time.Duration) {
	if duration <= 0 {
		q.WeightQueue.AddWeight(item, weight)
		return
	}
	k := q.clock.Now().Add(duration).UnixNano()

	q.mut.Lock()
	if weight <= 0 {
		q.heap.Push(k, item)
	} else {
		if q.heaps[weight] == nil {
			q.heaps[weight] = heap.NewHeap[int64, T]()
		}
		q.heaps[weight].Push(k, item)
	}
	q.mut.Unlock()

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

func (q *weightDelayingQueue[T]) loopWorker() {
	for {
		t, weight, ok, next := q.next()
		if ok {
			q.WeightQueue.AddWeight(t, weight)
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

func (q *weightDelayingQueue[T]) next() (t T, weight int, ok bool, wait *time.Duration) {
	q.mut.Lock()
	defer q.mut.Unlock()

	var waitDuration *time.Duration

	now := q.clock.Now()
	// The highest weight queue is always checked first
	if k, v, ok := q.heap.Peek(); ok {
		d := time.Unix(0, k).Sub(now)
		if d <= 0 {
			if q.heap.Remove(v) {
				return v, 0, true, nil
			}
		} else if waitDuration == nil || d < *waitDuration {
			waitDuration = &d
		}
	}

	// Check if the orders slice is out of sync with the queues map
	if len(q.orders) != len(q.heaps) {
		orders := maps.Keys(q.heaps)
		sort.Ints(orders)
		q.orders = orders
	}

	for i := len(q.orders) - 1; i >= 0; i-- {
		weight := q.orders[i]
		h := q.heaps[weight]
		k, v, ok := h.Peek()
		if !ok {
			continue
		}

		d := time.Unix(0, k).Sub(now)
		if d <= 0 {
			if h.Remove(v) {
				return v, weight, true, nil
			}
		} else if waitDuration == nil || d < *waitDuration {
			waitDuration = &d
		}
	}

	return t, 0, false, waitDuration
}

func (q *weightDelayingQueue[T]) Cancel(item T) bool {
	q.mut.Lock()
	defer q.mut.Unlock()

	deleted := q.heap.Remove(item)
	for _, h := range q.heaps {
		if h.Remove(item) {
			deleted = true
		}
	}
	return deleted
}
