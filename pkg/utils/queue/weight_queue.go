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

	"sigs.k8s.io/kwok/pkg/utils/maps"
)

// WeightQueue is a generic weight queue interface.
type WeightQueue[T any] interface {
	Queue[T]
	weightQueueInterface[T]
}

type weightQueueInterface[T any] interface {
	// AddWeight adds an item to the queue with the given weight.
	// if weight is zero, it is the highest weight.
	// otherwise, the weight ranges from 1 to n, with higher numbers meaning higher weight.
	AddWeight(item T, weight int)
}

type weightQueue[T any] struct {
	queue  Queue[T]
	queues map[int]Queue[T]
	orders []int

	signal chan struct{}
	mut    sync.RWMutex
}

// NewWeightQueue returns a new WeightQueue.
func NewWeightQueue[T any]() WeightQueue[T] {
	q := &weightQueue[T]{
		queue:  NewQueue[T](),
		queues: map[int]Queue[T]{},
		signal: make(chan struct{}, 1),
	}
	return q
}

func (q *weightQueue[T]) Add(item T) {
	q.queue.Add(item)

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

func (q *weightQueue[T]) AddWeight(item T, weight int) {
	if weight <= 0 {
		q.Add(item)
		return
	}

	q.mut.Lock()
	if q.queues[weight] == nil {
		q.queues[weight] = NewQueue[T]()
	}
	q.queues[weight].Add(item)
	q.mut.Unlock()

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

func (q *weightQueue[T]) step() bool {
	q.mut.Lock()
	defer q.mut.Unlock()

	// Check if the orders slice is out of sync with the queues map
	if len(q.orders) != len(q.queues) {
		orders := maps.Keys(q.queues)
		sort.Ints(orders)
		q.orders = orders
	}

	var added bool
	for i := len(q.orders) - 1; i >= 0; i-- {
		weight := q.orders[i]
		queue := q.queues[weight]
		times := weight
		for j := 0; j != times; j++ {
			t, ok := queue.Get()
			if !ok {
				break
			}

			q.queue.Add(t)
			added = true
		}
	}
	return added
}

func (q *weightQueue[T]) Get() (T, bool) {
	t, ok := q.queue.Get()
	if ok {
		return t, ok
	}

	if q.step() {
		return q.queue.Get()
	}
	return t, false
}

func (q *weightQueue[T]) GetOrWait() T {
	t, ok := q.Get()
	if ok {
		return t
	}

	// Wait for an item to be added.
	for range q.signal {
		t, ok = q.Get()
		if ok {
			return t
		}
	}
	panic("unreachable")
}

func (q *weightQueue[T]) GetOrWaitWithDone(done <-chan struct{}) (T, bool) {
	t, ok := q.Get()
	if ok {
		return t, ok
	}

	// Wait for an item to be added.
	for {
		select {
		case <-done:
			return t, false
		case <-q.signal:
			t, ok = q.Get()
			if ok {
				return t, true
			}
		}
	}
}

func (q *weightQueue[T]) Len() int {
	size := q.queue.Len()

	q.mut.RLock()
	defer q.mut.RUnlock()
	for _, queue := range q.queues {
		size += queue.Len()
	}
	return size
}
