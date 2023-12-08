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

// MapWithError returns a new slice containing the results of applying the given function
// to all elements in the slice that satisfy the predicate f.
func MapWithError[S ~[]T, T any, O any](s S, f func(T) (O, error)) ([]O, error) {
	out := make([]O, len(s))
	for i := range s {
		o, err := f(s[i])
		if err != nil {
			return nil, err
		}
		out[i] = o
	}
	return out, nil
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

// FilterAndMap returns a new slice containing the results of applying the given function
// to all elements in the slice that satisfy the predicate f.
func FilterAndMap[S ~[]T, T any, O any](s S, f func(T) (O, bool)) []O {
	out := make([]O, 0, len(s))
	for _, v := range s {
		if o, ok := f(v); ok {
			out = append(out, o)
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

// Unique returns a new slice containing only the unique elements in the slice.
func Unique[S ~[]T, T comparable](s S) []T {
	if len(s) <= 1 {
		return s
	}
	exist := make(map[T]struct{})
	out := make([]T, 0, len(s))
	for _, v := range s {
		if _, ok := exist[v]; !ok {
			exist[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Equal returns true if the two slices are equal.
func Equal[S ~[]T, T comparable](s1, s2 S) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

// Reverse returns a new slice containing the elements of the slice in reverse order.
func Reverse[S ~[]T, T any](s S) []T {
	out := make([]T, len(s))
	for i := range s {
		out[len(s)-1-i] = s[i]
	}
	return out
}

// GroupBy returns a map of slices grouped by the given function.
func GroupBy[S ~[]T, T any, K comparable](s S, f func(T) K) map[K][]T {
	out := make(map[K][]T)
	for _, v := range s {
		k := f(v)
		out[k] = append(out[k], v)
	}
	return out
}
