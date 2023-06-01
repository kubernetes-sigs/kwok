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

package format

import (
	"fmt"
	"time"
)

// HumanDuration returns a succinct representation of the provided duration
// with limited precision for consumption by humans. It provides ~2-3 significant
// figures of duration.
func HumanDuration(d time.Duration) string {
	// Allow deviation no more than 2 seconds(excluded) to tolerate machine time
	// inconsistence, it can be considered as almost now.
	switch {
	case d <= -2*time.Second:
		return "<invalid>"
	case d < time.Millisecond:
		return "0s"
	}

	milliseconds := int(d / time.Millisecond)
	switch {
	case milliseconds < 100:
		return fmt.Sprintf("%dms", milliseconds)
	case milliseconds < 1000:
		return fmt.Sprintf("0.%ds", milliseconds/100)
	}

	seconds := int(d / time.Second)
	switch {
	case seconds < 10:
		ms := int(d/time.Millisecond) % 1000
		if ms < 100 {
			return fmt.Sprintf("%ds", seconds)
		}
		return fmt.Sprintf("%d.%ds", seconds, ms/100)
	case seconds < 60*2:
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := int(d / time.Minute)
	switch {
	case minutes < 10:
		s := int(d/time.Second) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, s)
	case minutes < 60*3:
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d / time.Hour)
	switch {
	case hours < 8:
		m := int(d/time.Minute) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, m)
	case hours < 48:
		return fmt.Sprintf("%dh", hours)
	case hours < 24*8:
		h := hours % 24
		if h == 0 {
			return fmt.Sprintf("%dd", hours/24)
		}
		return fmt.Sprintf("%dd%dh", hours/24, h)
	case hours < 24*365*2:
		return fmt.Sprintf("%dd", hours/24)
	case hours < 24*365*8:
		dy := (hours / 24) % 365
		if dy == 0 {
			return fmt.Sprintf("%dy", hours/24/365)
		}
		return fmt.Sprintf("%dy%dd", hours/24/365, dy)
	}

	return fmt.Sprintf("%dy", hours/24/365)
}
