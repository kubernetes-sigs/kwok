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

package wait

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type conditionMocker struct {
	expectElapse      time.Duration
	expectUndoneCount int
	expectErrorCount  int

	undoneCount int
	errorCount  int
}

func (c *conditionMocker) condition(ctx context.Context) (bool, error) {
	if c.expectElapse > 0 {
		time.Sleep(c.expectElapse)
	}

	if c.undoneCount < c.expectUndoneCount {
		c.undoneCount++
		return false, nil
	}

	if c.errorCount < c.expectErrorCount {
		c.errorCount++
		return false, fmt.Errorf("error times %d", c.errorCount)
	}

	return true, nil
}

func TestPoll_Default(t *testing.T) {
	cm := &conditionMocker{}

	err := Poll(context.Background(), cm.condition, WithTimeout(100*time.Millisecond), WithInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("Poll() error = %v, wantErr %v", err, false)
	}
}

func TestPoll_WithTimeout(t *testing.T) {
	cm := &conditionMocker{
		expectElapse: time.Second,
	}

	timeout := 50 * time.Millisecond

	start := time.Now()
	err := Poll(context.Background(), cm.condition, WithTimeout(timeout))
	elapsed := time.Since(start)

	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Poll() error = %v, want %v", err, context.DeadlineExceeded)
	}

	if elapsed < timeout {
		t.Fatalf("Poll() timeout = %v, want >= %v", elapsed, timeout)
	}

	if elapsed >= cm.expectElapse {
		t.Fatalf("Poll() timeout = %v, want < %v", elapsed, cm.expectElapse)
	}
}

func TestPoll_WithInterval(t *testing.T) {
	cm := &conditionMocker{
		expectUndoneCount: 3,
	}

	interval := 10 * time.Millisecond

	start := time.Now()
	err := Poll(context.Background(), cm.condition, WithInterval(interval))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Poll() error = %v, wantErr %v", err, false)
	}

	if cm.undoneCount != cm.expectUndoneCount {
		t.Fatalf("Poll() called condition %d times, want %d times", cm.undoneCount, cm.expectUndoneCount)
	}

	expectedMinElapsed := interval * time.Duration(cm.expectUndoneCount-1)
	if elapsed < expectedMinElapsed {
		t.Fatalf("Poll() elapsed = %v, want >= %v", elapsed, expectedMinElapsed)
	}
}

func TestPoll_WithImmediate(t *testing.T) {
	// Define a condition function that returns true immediately when WithImmediate is set
	conditionImmediate := func(ctx context.Context) (bool, error) {
		return true, nil
	}

	// Define a condition function that simulates a delay and returns true
	conditionDelayed := func(ctx context.Context) (bool, error) {
		time.Sleep(100 * time.Millisecond) // Simulate a delay
		return true, nil
	}

	// Test with immediate condition
	startImmediate := time.Now()
	errImmediate := Poll(context.Background(), conditionImmediate, WithImmediate())
	elapsedImmediate := time.Since(startImmediate)

	if errImmediate != nil {
		t.Fatalf("Poll() with immediate condition failed: %v", errImmediate)
	}

	if elapsedImmediate >= defaultPollInterval {
		t.Fatalf("Poll() elapsed = %v, want < %v", elapsedImmediate, defaultPollInterval)
	}

	// Test with delayed condition
	startDelayed := time.Now()
	errDelayed := Poll(context.Background(), conditionDelayed)
	elapsedDelayed := time.Since(startDelayed)

	if errDelayed != nil {
		t.Fatalf("Poll() with delayed condition failed: %v", errDelayed)
	}

	if elapsedDelayed < defaultPollInterval {
		t.Fatalf("Poll() elapsed = %v, want >= %v", elapsedDelayed, defaultPollInterval)
	}
}

func TestPoll_WithExponentialBackoff(t *testing.T) {
	backoff := &Backoff{
		Duration: 10 * time.Millisecond,
		Factor:   2.0,
		Steps:    3,
	}
	count := 0
	condition := func(ctx context.Context) (bool, error) {
		count++
		if count == 3 {
			return true, nil
		}
		return false, nil
	}

	start := time.Now()
	err := Poll(context.Background(), condition, WithExponentialBackoff(backoff))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Poll() error = %v, wantErr %v", err, false)
	}

	expectedMinElapsed := backoff.Duration * 3
	if elapsed < expectedMinElapsed {
		t.Fatalf("Poll() elapsed = %v, want >= %v", elapsed, expectedMinElapsed)
	}
}
func TestPoll_EdgeCases(t *testing.T) {
	cm := &conditionMocker{}

	// Zero interval
	err := Poll(context.Background(), cm.condition, WithInterval(0))
	if err != nil {
		t.Fatalf("Poll() error = %v, wantErr %v", err, false)
	}

	// Zero timeout
	err = Poll(context.Background(), cm.condition, WithTimeout(0))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Poll() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestWithContinueOnError(t *testing.T) {
	opts := &Options{}
	fn := WithContinueOnError(3)
	fn(opts)
	if opts.ContinueOnError != 3 {
		t.Fatalf("WithContinueOnError() = %v, want %v", opts.ContinueOnError, 3)
	}
}

func TestWithTimeout(t *testing.T) {
	opts := &Options{}
	timeout := 30 * time.Second
	fn := WithTimeout(timeout)
	fn(opts)
	if opts.Timeout != timeout {
		t.Fatalf("WithTimeout() = %v, want %v", opts.Timeout, timeout)
	}
}

func TestWithInterval(t *testing.T) {
	opts := &Options{}
	interval := 5 * time.Second
	fn := WithInterval(interval)
	fn(opts)
	if opts.Interval != interval {
		t.Fatalf("WithInterval() = %v, want %v", opts.Interval, interval)
	}
}

func TestWithImmediate(t *testing.T) {
	opts := &Options{}
	fn := WithImmediate()
	fn(opts)
	if !opts.Immediate {
		t.Fatalf("WithImmediate() = %v, want %v", opts.Immediate, true)
	}
}

func TestWithExponentialBackoff(t *testing.T) {
	opts := &Options{}
	backoff := &Backoff{
		Duration: 1 * time.Second,
		Factor:   2.0,
		Steps:    3,
	}
	fn := WithExponentialBackoff(backoff)
	fn(opts)
	if opts.Backoff != backoff {
		t.Fatalf("WithExponentialBackoff() = %v, want %v", opts.Backoff, backoff)
	}
}

func TestJitter(t *testing.T) {
	duration := 1 * time.Second
	maxFactor := 1.0
	result := Jitter(duration, maxFactor)
	if result < duration || result > 2*duration {
		t.Fatalf("Jitter() = %v, want between %v and %v", result, duration, 2*duration)
	}
}

func TestPoll_WithContinueOnError(t *testing.T) {
	cm := &conditionMocker{
		expectErrorCount: 2,
	}

	// Define options with ContinueOnError set to 2
	opts := []OptionFunc{
		WithContinueOnError(2),
	}

	err := Poll(context.Background(), cm.condition, opts...)

	if err == nil || err.Error() != "error times 2" {
		t.Fatalf("Poll() error = %v, want %v", err, "error times 2")
	}

	if cm.errorCount != cm.expectErrorCount {
		t.Fatalf("Poll() called condition %d times, want %d times", cm.errorCount, cm.expectErrorCount)
	}
}
