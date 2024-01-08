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
	"reflect"
	"testing"
)

func TestWeightQueue(t *testing.T) {
	pq := NewWeightQueue[string]()

	pq.AddWeight("1", 1)
	pq.AddWeight("2", 1)
	pq.AddWeight("3", 1)
	pq.AddWeight("4", 2)
	pq.AddWeight("5", 2)
	pq.AddWeight("6", 2)
	pq.AddWeight("7", 4)
	pq.AddWeight("8", 0)

	list := []string{}
	for pq.Len() != 0 {
		item := pq.GetOrWait()
		list = append(list, item)
	}

	want := []string{"8", "7", "4", "5", "1", "6", "2", "3"}
	if len(list) != len(want) {
		t.Errorf("got %v, want %v", list, want)
	}

	if !reflect.DeepEqual(list, want) {
		t.Errorf("got %v, want %v", list, want)
	}
}
