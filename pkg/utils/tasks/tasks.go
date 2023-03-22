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
	"sync"
	"time"

	"github.com/wzshiming/llrb"
)

// ParallelPriorityTasks is a task queue that can be executed in parallel with priority
type ParallelPriorityTasks struct {
	mut           sync.RWMutex
	tasks         *llrb.Tree[int, chan func()]
	parallelTasks *ParallelTasks
	arouse        chan struct{}
	n             int
}

// NewParallelPriorityTasks create a new ParallelPriorityTasks
func NewParallelPriorityTasks(n int) *ParallelPriorityTasks {
	if n <= 0 {
		n = 1
	}
	return &ParallelPriorityTasks{
		tasks:         llrb.NewTree[int, chan func()](),
		parallelTasks: NewParallelTasks(n),
		n:             n,
	}
}

// Add adds a task to the queue with priority.
// priority is the priority of the task, the higher the priority, the earlier the execution
// priority > 0, Perform a number of tasks at a time equal to the priority level.
// priority == 0, Perform one task at a time
// priority < 0, If there is a high priority task, the lower priority task will never be executed.
func (p *ParallelPriorityTasks) Add(priority int, fun func()) {
	keyPriority := -priority
	p.mut.RLock()
	ch, _ := p.tasks.Get(keyPriority)
	p.mut.RUnlock()
	if ch == nil {
		p.mut.Lock()
		ch, _ = p.tasks.Get(keyPriority)
		if ch == nil {
			length := priority
			if length <= 0 {
				length = 1
			} else if length > p.n {
				length = p.n
			}
			ch = make(chan func(), length)
			p.tasks.Put(keyPriority, ch)
		}

		if p.arouse == nil {
			p.arouse = make(chan struct{}, 1)
			go p.run()
		}
		p.mut.Unlock()
	}

	ch <- fun
	p.arouseTask()
}

func (p *ParallelPriorityTasks) arouseTask() {
	select {
	case p.arouse <- struct{}{}:
	default:
	}
}

func (p *ParallelPriorityTasks) run() {
	for {
		count := p.runStep()
		if count == 0 {
			select {
			case <-p.arouse:
			case <-time.After(time.Second):
			}
		}
	}
}

func (p *ParallelPriorityTasks) runStep() int {
	p.mut.RLock()
	defer p.mut.RUnlock()
	var count int
	p.tasks.Range(func(keyPriority int, ch chan func()) bool {
		priority := -keyPriority

		if priority > 0 {
			// Perform a number of tasks at a time equal to the priority level.
			for i := 0; i != priority; i++ {
				select {
				case fun := <-ch:
					count++
					p.parallelTasks.Add(fun)
				default:
					return true
				}
			}
		} else {
			select {
			case fun := <-ch:
				count++
				p.parallelTasks.Add(fun)
				return false
			default:
			}

			// If there is a high priority task, the lower priority task will never be executed.
			if count > 0 {
				return false
			}
		}
		return true
	})
	return count
}

func (p *ParallelPriorityTasks) wait() {
	p.parallelTasks.wait()
}

// ParallelTasks is a task queue that can be executed in parallel
type ParallelTasks struct {
	wg     sync.WaitGroup
	bucket chan struct{}
	tasks  chan func()
}

// NewParallelTasks create a new ParallelTasks
func NewParallelTasks(n int) *ParallelTasks {
	if n <= 0 {
		n = 1
	}
	return &ParallelTasks{
		bucket: make(chan struct{}, n),
		tasks:  make(chan func()),
	}
}

// Add add a task to the queue
func (p *ParallelTasks) Add(fun func()) {
	p.wg.Add(1)
	select {
	case p.tasks <- fun: // there are idle threads
	case p.bucket <- struct{}{}: // there are free threads
		go p.fork()
		p.tasks <- fun
	}
}

func (p *ParallelTasks) fork() {
	defer func() {
		<-p.bucket
	}()
	timer := time.NewTimer(time.Second / 2)
	for {
		select {
		case <-timer.C: // idle threads
			return
		case fun := <-p.tasks:
			fun()
			p.wg.Done()
			timer.Reset(time.Second / 2)
		}
	}
}

func (p *ParallelTasks) wait() {
	p.wg.Wait()
}
