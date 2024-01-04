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

package backoff

import (
	"sync"
	"time"

	"sigs.k8s.io/kwok/pkg/utils/wait"
)

const (
	DefaultInitialInterval time.Duration = 10 * time.Second
	DefaultMaxInterval     time.Duration = 5 * time.Minute
	DefaultFactor          float64       = 2.0
	DefaultJitter          float64       = 0.1
)

// Set is used for tracking multiple items that shares a same backoff setting.
// Set is inspired by k8s.io/client-go/util/flowcontrol/backoff.go.
// In addition to re implementing it with in a generic style, we also simplify
// the functionality as much as possible according to our own needs.
type Set[T comparable] struct {
	sync.RWMutex
	// the init backoff period of a new added item is initialInterval plus jitter
	initialInterval time.Duration
	// Factor is the multiplying factor for each revision on the backoff duration.
	factor float64
	// The (revised) backoff duration is capped at cap.
	cap time.Duration
	// Jitter adds a randomized amount of time to the revised duration.
	// The amount is chosen uniformly at random from the interval between
	// zero and `jitter*current revised duration`.
	// No jitter will be added if jitter is zero.
	jitter float64
	// entries holds the current backoff duration of all the items.
	entries map[T]time.Duration
}

// SetOptions configures the Set
type SetOptions struct {
	initialInterval time.Duration
	factor          float64
	maxInterval     time.Duration
	jitter          float64
}

// OptionFunc configures a SetOption
type OptionFunc func(*SetOptions)

// WithInitialInterval configures the initialInterval for the backoff.
// DefaultInitialInterval will be used instead if given a negative augment.
func WithInitialInterval(interval time.Duration) OptionFunc {
	if interval < 0 {
		interval = DefaultInitialInterval
	}
	return func(options *SetOptions) {
		options.initialInterval = interval
	}
}

// WithMaxInterval limits the max period for the backoff.
// DefaultMaxInterval will be used instead if given a non-positive augment.
func WithMaxInterval(maxInterval time.Duration) OptionFunc {
	if maxInterval <= 0 {
		maxInterval = DefaultMaxInterval
	}
	return func(opt *SetOptions) {
		opt.maxInterval = maxInterval
	}
}

// WithFactor configures the factor for the backoff.
// DefaultFactor will be used instead if given a non-positive augment.
func WithFactor(factor float64) OptionFunc {
	if factor <= 0.0 {
		factor = DefaultFactor
	}
	return func(opt *SetOptions) {
		opt.factor = factor
	}
}

// WithJitter configures the jitter for the backoff.
// DefaultJitter will be used instead if given a negative augment.
func WithJitter(jitter float64) OptionFunc {
	if jitter < 0.0 {
		jitter = DefaultJitter
	}
	return func(opt *SetOptions) {
		opt.jitter = jitter
	}
}

func defaultOption() *SetOptions {
	return &SetOptions{
		initialInterval: DefaultInitialInterval,
		factor:          DefaultFactor,
		maxInterval:     DefaultMaxInterval,
		jitter:          DefaultJitter,
	}
}

// NewBackoffSet creates an instance of Set using the given option function augments.
func NewBackoffSet[T comparable](opts ...OptionFunc) *Set[T] {
	o := defaultOption()

	for _, optFunc := range opts {
		optFunc(o)
	}

	return &Set[T]{
		initialInterval: o.initialInterval,
		factor:          o.factor,
		cap:             o.maxInterval,
		jitter:          o.jitter,
		entries:         make(map[T]time.Duration),
	}
}

// Get returns the current backoff period of an item if it exists.
func (p *Set[T]) Get(item T) (time.Duration, bool) {
	p.RLock()
	defer p.RUnlock()

	entry, ok := p.entries[item]
	if ok {
		return entry, ok
	}
	return 0, false
}

// Add adds an item to the Set and returns the initial backoff period if it does not exist.
func (p *Set[T]) Add(item T) (time.Duration, bool) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.entries[item]; !ok {
		return 0, false
	}

	duration := p.addUnsafe(item)
	return duration, true
}

// AddOrUpdate initializes an item if it does not exist
// or revises its backoff period if the item already exist.
func (p *Set[T]) AddOrUpdate(item T) time.Duration {
	p.Lock()
	defer p.Unlock()

	_, ok := p.entries[item]
	if !ok {
		return p.addUnsafe(item)
	}

	nextDuration := p.stepUnsafe(item)
	return nextDuration
}

// Step revises and returns the backoff period of an item if it exists.
func (p *Set[T]) Step(item T) (time.Duration, bool) {
	p.Lock()
	defer p.Unlock()

	_, ok := p.entries[item]
	if !ok {
		return 0, false
	}

	nextDuration := p.stepUnsafe(item)
	return nextDuration, true
}

// stepUnsafe revises and returns the next backoff period of an item.
// The caller must ensure the item already exist and take a lock before calling it.
func (p *Set[T]) stepUnsafe(item T) time.Duration {
	entry := p.entries[item]
	p.entries[item] = p.delay(entry)
	return p.entries[item]
}

// stepUnsafe adds(resets) an item and initialize its backoff period with the initialInterval plus jitter.
// The caller must take a lock before calling it.
func (p *Set[T]) addUnsafe(item T) time.Duration {
	if p.jitter > 0 {
		p.entries[item] = wait.Jitter(p.initialInterval, p.jitter)
	} else {
		p.entries[item] = p.initialInterval
	}
	return p.entries[item]
}

// delay calculates the next backoff period based on the given current period.
func (p *Set[T]) delay(duration time.Duration) (next time.Duration) {
	// revise the backoff duration by multiplying the factor
	next = time.Duration(float64(duration) * p.factor)

	// add jitter
	if p.jitter > 0 {
		next = wait.Jitter(next, p.jitter)
	}
	// cap the revised backoff duration
	next = min(next, p.cap)

	return next
}

// Remove removes the item from the Set.
func (p *Set[T]) Remove(item T) {
	delete(p.entries, item)
}
