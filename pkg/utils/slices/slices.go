/*
Copyright 2022 The Kubernetes Authors.

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

package slices

// Map returns a new slice containing the results of applying the given function
func Map[S ~[]T, T any, O any](s S, f func(T) O) []O {
	out := make([]O, len(s))
	for i := range s {
		out[i] = f(s[i])
	}
	return out
}

// Find returns the first element in the slice that satisfies the predicate f.
func Find[S ~[]T, T any](s S, f func(T) bool) (t T, ok bool) {
	for _, v := range s {
		if f(v) {
			return v, true
		}
	}
	return t, false
}

// Filter returns a new slice containing all elements in the slice that satisfy the predicate f.
func Filter[S ~[]T, T any](s S, f func(T) bool) []T {
	out := make([]T, 0, len(s))
	for _, v := range s {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}

// Contains returns true if the slice contains the given element.
func Contains[S ~[]T, T comparable](s S, t T) bool {
	for _, v := range s {
		if v == t {
			return true
		}
	}
	return false
}
