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

package log

import (
	"context"
	"time"

	"k8s.io/utils/clock"
)

// Elapsed logs a message if the elapsed time is greater than the given duration.
func Elapsed(ctx context.Context, c clock.Clock, level Level, duration time.Duration, msg string) func() {
	start := c.Now()
	return func() {
		elapsed := c.Since(start)
		if elapsed >= duration {
			logger := FromContext(ctx)
			logger.Log(level, msg,
				"elapsed", elapsed,
			)
		}
	}
}
