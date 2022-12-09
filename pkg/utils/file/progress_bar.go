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

package file

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type progressBar struct {
	total          int
	current        int
	lastUpdateTime time.Time
	startTime      time.Time
}

func newProgressBar() *progressBar {
	return &progressBar{
		startTime: time.Now(),
	}
}

func (p *progressBar) Update(current, total int) {
	p.current = current
	p.total = total
}

func (p *progressBar) Print() {
	if p.total == 0 {
		return
	}
	now := time.Now()
	if p.current < p.total &&
		now.Sub(p.lastUpdateTime) < time.Second/10 {
		return
	}
	p.lastUpdateTime = now

	if p.current >= p.total {
		_, _ = fmt.Fprintf(os.Stderr, "\r%-60s| 100%%  %-5s\n", strings.Repeat("#", 60), time.Since(p.startTime).Truncate(time.Second))
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "\r%-60s| %.1f%% %-5s", strings.Repeat("#", int(float64(p.current)/float64(p.total)*60)), float64(p.current)/float64(p.total)*100, time.Since(p.startTime).Truncate(time.Second))
	}
}
