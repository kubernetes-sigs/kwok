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
	"sync"
	"time"

	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/utils/backoff"
)

// RetryQueue abstracts the behaviour of a queue for retry purpose.
// Compared to a general Queue type, the meaning of the methods of
// a retry queue is re-defined.
// The caller calls Get to peek an item to retry, but the item should
// not be removed from the underlying state unless the
// caller explicitly calls the Remove method. The caller MUST also
// explicitly re-Add the item to the queue if a retry fails or Remove it
// if it succeeds.
type RetryQueue[T comparable] interface {
	Queue[T]
	// Remove excludes an item from being tracked.
	Remove(T)

	// Has tells whether an item is already being tracked.
	Has(T) bool
}

// backoffRetryQueue implements RetryQueue based on a backoff mechanism.
// An item can be taken out of the queue (for retry) only by waiting for
// some amount of delay time. The caller MUST tell the queue that a retry
// fails by explicitly re pushing the item to the queue, leading its next
// waiting time to increase (exponentially).
//
// In terms of implementation, the delaying queue serves as the skeleton of
// the retry queue to play the role of waiting-before-getting.
// The backoffSet, as an auxiliary structure, helps track the backoff delay
// time of all the items.
type backoffRetryQueue[T comparable] struct {
	sync.RWMutex

	DelayingQueue[T]

	backoffSet *backoff.Set[T]
}

// NewDefaultBackoffRetryQueue creates an instance of backoffRetryQueue using the default options.
func NewDefaultBackoffRetryQueue[T comparable]() RetryQueue[T] {
	realClock := clock.RealClock{}
	return &backoffRetryQueue[T]{
		DelayingQueue: NewDelayingQueue[T](realClock),
		backoffSet:    backoff.NewBackoffSet[T](),
	}
}

// NewBackoffRetryQueue creates an instance of backoffRetryQueue with the given options.
func NewBackoffRetryQueue[T comparable](clock clock.Clock,
	initialInterval, maxInterval time.Duration, factor, jitter float64) RetryQueue[T] {
	backoffSet := backoff.NewBackoffSet[T](
		backoff.WithInitialInterval(initialInterval),
		backoff.WithMaxInterval(maxInterval),
		backoff.WithJitter(jitter),
		backoff.WithFactor(factor),
	)

	return &backoffRetryQueue[T]{
		DelayingQueue: NewDelayingQueue[T](clock),
		backoffSet:    backoffSet,
	}
}

// Add initializes (or revises) an item in the underlying backoffSet (if it already exists)
// and pushes it to the delaying queue with the (revised) backoff period of the item.
// Adding an item that already exits in the backoffSet leads the latency to increase
// exponentially the next time it is popped from the queue.
// The caller SHOULD NOT Add an item to the queue before popping it out. This may
// lead to unexpected behaviour.
func (q *backoffRetryQueue[T]) Add(item T) {
	q.Lock()
	defer q.Unlock()

	delay := q.backoffSet.AddOrUpdate(item)
	q.AddAfter(item, delay)
}

// Remove removes an item from both the underlying backoffSet and the delaying queue.
func (q *backoffRetryQueue[T]) Remove(item T) {
	q.Lock()
	defer q.Unlock()

	q.backoffSet.Remove(item)
	q.Cancel(item)
}

// Has tells whether an item exists in the backoffSet.
func (q *backoffRetryQueue[T]) Has(item T) bool {
	q.RLock()
	defer q.Unlock()

	_, ok := q.backoffSet.Get(item)
	return ok
}
