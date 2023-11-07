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

package sets

// Sets is a set of items.
type Sets[T comparable] map[T]struct{}

// NewSets creates a Sets from a list of values.
func NewSets[T comparable](items ...T) Sets[T] {
	s := make(Sets[T])
	s.Insert(items...)
	return s
}

// Insert adds items to the set.
func (s Sets[T]) Insert(items ...T) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

// Delete removes all items from the set.
func (s Sets[T]) Delete(items ...T) {
	for _, item := range items {
		delete(s, item)
	}
}

// Has returns true if and only if item is contained in the set.
func (s Sets[T]) Has(item T) bool {
	_, contained := s[item]
	return contained
}

// Len returns the size of the set.
func (s Sets[T]) Len() int {
	return len(s)
}

// Clear empties the set.
func (s Sets[T]) Clear() {
	for key := range s {
		delete(s, key)
	}
}
