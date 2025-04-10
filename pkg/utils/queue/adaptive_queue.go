/*
Copyright 2025 The Kubernetes Authors.

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
	"context"
	"sync"
	"time"
)

type AdaptiveQueue[T any] struct {
	ctx         context.Context
	startFunc   func(ctx context.Context)
	latestStart time.Time
	mut         sync.Mutex
	queue       Queue[T]
	count       int
}

func NewAdaptiveQueue[T any](ctx context.Context, q Queue[T], startFunc func(ctx context.Context)) *AdaptiveQueue[T] {
	return &AdaptiveQueue[T]{
		ctx:         ctx,
		startFunc:   startFunc,
		latestStart: time.Now(),
		queue:       q,
	}
}

func (p *AdaptiveQueue[T]) GetOrWaitWithDone(done <-chan struct{}) (T, bool) {
	t, ok := p.queue.GetOrWaitWithDone(done)
	if !ok {
		return t, false
	}

	length := p.queue.Len()
	if length > p.count/10+1 {
		p.mut.Lock()
		defer p.mut.Unlock()
		now := time.Now()
		sub := now.Sub(p.latestStart)

		if sub >= time.Duration(p.count)*time.Millisecond {
			go p.startFunc(p.ctx)
			p.latestStart = now
			p.count++
		}
	}
	return t, true
}
