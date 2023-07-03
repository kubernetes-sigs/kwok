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

type staticGetter[T any] struct {
	data T
}

// NewStaticGetter returns a new Getter that returns the given list.
func NewStaticGetter[T any](data T) Getter[T] {
	return &staticGetter[T]{data: data}
}

func (s *staticGetter[T]) Get() T {
	return s.data
}

func (s *staticGetter[T]) Version() string {
	return ""
}
