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

package maps

import (
	"sync"
)

// SyncMap is a wrapper around sync.Map that provides a few additional methods.
type SyncMap[K comparable, V any] struct {
	m sync.Map
}

// Load returns the value stored in the map for a key,
// or nil if no  value is present.
func (m *SyncMap[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, false
	}
	return v.(V), true
}

// Store sets the value for a key.
func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Delete deletes the value for a key.
func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Range calls f sequentially for each key and value present in the map.
func (m *SyncMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(K), value.(V))
	})
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
func (m *SyncMap[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}
	return v.(V), loaded
}

// LoadOrStore returns the existing value for the key if present.
func (m *SyncMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	if !loaded {
		return value, loaded
	}
	return v.(V), loaded
}

// Swap stores value for key and returns the previous value for that key.
func (m *SyncMap[K, V]) Swap(key K, value V) (V, bool) {
	v, loaded := m.m.Swap(key, value)
	if !loaded {
		return value, loaded
	}
	return v.(V), loaded
}

// Size returns the number of items in the map.
func (m *SyncMap[K, V]) Size() int {
	size := 0
	m.m.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

// Keys returns all the keys in the map.
func (m *SyncMap[K, V]) Keys() []K {
	keys := []K{}
	m.m.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(K))
		return true
	})
	return keys
}

// Values returns all the values in the map.
func (m *SyncMap[K, V]) Values() []V {
	values := []V{}
	m.m.Range(func(key, value interface{}) bool {
		values = append(values, value.(V))
		return true
	})
	return values
}

// IsEmpty returns true if the map is empty.
func (m *SyncMap[K, V]) IsEmpty() bool {
	empty := true
	m.m.Range(func(key, value interface{}) bool {
		empty = false
		return false
	})
	return empty
}
