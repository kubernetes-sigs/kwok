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

package resources

import (
	"sync"
)

type cacheGetter[O any] struct {
	getter Getter[O]

	currentVer string
	data       O

	mut sync.RWMutex
}

func withCache[O any](getter Getter[O]) Getter[O] {
	return &cacheGetter[O]{getter: getter}
}

func (g *cacheGetter[O]) Get() O {
	g.mut.RLock()
	latestVer := g.getter.Version()
	if g.currentVer == latestVer {
		data := g.data
		g.mut.RUnlock()
		return data
	}
	g.mut.RUnlock()

	g.mut.Lock()
	defer g.mut.Unlock()
	if g.currentVer == latestVer {
		data := g.data
		return data
	}

	data := g.getter.Get()
	g.data = data
	g.currentVer = latestVer
	return data
}

func (g *cacheGetter[O]) Version() string {
	return g.getter.Version()
}
