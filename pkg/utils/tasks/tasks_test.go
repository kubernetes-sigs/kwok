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

package tasks

import (
	"reflect"
	"testing"
	"time"
)

func TestParallelPriorityTasks(t *testing.T) {
	tasks := NewParallelPriorityTasks(1)

	ch := make(chan int, 10)
	for i := 0; i < 10; i++ {
		i := i
		tasks.Add(i, func() {
			ch <- i
		})
	}

	time.Sleep(1 * time.Second)
	tasks.wait()
	close(ch)

	got := []int{}
	for o := range ch {
		got = append(got, o)
	}
	want := []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tasks did not complete in priority order: got %v, want %v", got, want)
	}
}

func TestParallelTasks(t *testing.T) {
	tasks := NewParallelTasks(4)
	startTime := time.Now()
	for i := 0; i < 10; i++ {
		tasks.Add(func() {
			time.Sleep(1 * time.Second)
		})
	}

	tasks.wait()
	elapsed := time.Since(startTime)
	if elapsed >= 4*time.Second {
		t.Fatalf("Tasks took too long to complete: %v", elapsed)
	} else if elapsed < 3*time.Second {
		t.Fatalf("Tasks completed too quickly: %v", elapsed)
	}
	t.Logf("Tasks completed in %v", elapsed)
}
