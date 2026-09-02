/*
Copyright 2026 The Kubernetes Authors.

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

package net

import (
	"context"
	"io"
	"runtime"
	"testing"
	"time"
)

// blockedStream is an io.ReadWriter whose Read blocks until unblock is called,
// which keeps a Tunnel copy goroutine in flight while the test cancels the
// context. It stands in for the network connections the real callers pass.
type blockedStream struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func newBlockedStream() *blockedStream {
	r, w := io.Pipe()
	return &blockedStream{r: r, w: w}
}

func (s *blockedStream) Read(p []byte) (int, error) {
	return s.r.Read(p)
}

func (s *blockedStream) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

// unblock releases a Read that is currently blocked, the way closing the
// underlying connection does.
func (s *blockedStream) unblock() {
	_ = s.r.Close()
}

// waitForGoroutines waits for the goroutine count to fall back to baseline.
// A leaked Tunnel copy goroutine is parked on a channel send and never exits,
// so it keeps the count above baseline until the deadline.
func waitForGoroutines(t *testing.T, baseline int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if n := runtime.NumGoroutine(); n <= baseline {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("goroutines did not return to baseline: got %d, want <= %d", n, baseline)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// runTunnel starts Tunnel in the background and returns a channel closed once
// it has returned.
func runTunnel(ctx context.Context, c1, c2 io.ReadWriter) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = Tunnel(ctx, c1, c2, make([]byte, 32), make([]byte, 32))
	}()
	return done
}

func TestTunnelDoesNotLeakWhenCanceledBeforeEitherCopyFinishes(t *testing.T) {
	c1 := newBlockedStream()
	c2 := newBlockedStream()

	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runTunnel(ctx, c1, c2)

	// Give both copies time to reach their blocking Read before canceling, so
	// that Tunnel returns through the outer ctx.Done() path.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	// Callers close the streams after Tunnel returns. That releases the copies,
	// and each one then reports its result on the error channel.
	c1.unblock()
	c2.unblock()

	waitForGoroutines(t, baseline, 3*time.Second)
}

func TestTunnelDoesNotLeakWhenCanceledAfterTheFirstCopyFinishes(t *testing.T) {
	c1 := newBlockedStream()
	c2 := newBlockedStream()

	// The c1 -> c2 copy ends immediately; the c2 -> c1 copy stays blocked, so
	// Tunnel receives one result and then returns through the inner
	// ctx.Done() path with the second still pending.
	c1.unblock()

	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := runTunnel(ctx, c1, c2)

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	c2.unblock()

	waitForGoroutines(t, baseline, 3*time.Second)
}
