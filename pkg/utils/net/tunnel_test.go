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
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"
)

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestTunnelReturnsCopyError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		writeErr bool
		reverse  bool
	}{
		{name: "first stream read"},
		{name: "second stream read", reverse: true},
		{name: "second stream write", writeErr: true},
		{name: "first stream write", writeErr: true, reverse: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			blocked := newBlockedStream()
			defer blocked.unblock()
			copyErr := errors.New("copy failed")
			var reader io.Reader = errorReader{err: copyErr}
			var writer io.Writer = io.Discard
			if tc.writeErr {
				reader = strings.NewReader("request")
				writer = errorWriter{err: copyErr}
			}
			c1 := struct {
				io.Reader
				io.Writer
			}{reader, io.Discard}
			c2 := struct {
				io.Reader
				io.Writer
			}{blocked, writer}
			done := make(chan error, 1)
			go func() {
				if tc.reverse {
					done <- Tunnel(ctx, c2, c1, nil)
				} else {
					done <- Tunnel(ctx, c1, c2, nil)
				}
			}()
			select {
			case err := <-done:
				if !errors.Is(err, copyErr) {
					t.Fatalf("Tunnel returned %v, want %v", err, copyErr)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Tunnel waited for the other stream after a copy error")
			}
		})
	}
}

func TestTunnelDrainsOtherStreamAfterEOF(t *testing.T) {
	ctx := t.Context()
	r, w := io.Pipe()
	defer func() {
		_ = r.Close()
		_ = w.Close()
	}()
	var response bytes.Buffer
	c1 := struct {
		io.Reader
		io.Writer
	}{strings.NewReader(""), &response}
	c2 := struct {
		io.Reader
		io.Writer
	}{r, io.Discard}
	done := make(chan error, 1)
	go func() {
		done <- Tunnel(ctx, c1, c2, nil)
	}()
	select {
	case err := <-done:
		t.Fatalf("Tunnel returned before the response: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := io.WriteString(w, "response"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Tunnel did not return after both streams reached EOF")
	}
	if response.String() != "response" {
		t.Fatalf("response = %q, want %q", response.String(), "response")
	}
}

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
		_ = Tunnel(ctx, c1, c2, nil)
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

// trackingBufferPool records when a buffer becomes available for reuse.
type trackingBufferPool struct {
	borrowed chan []byte
	released chan []byte
}

func (p *trackingBufferPool) Get() []byte {
	buf := make([]byte, 32)
	p.borrowed <- buf
	return buf
}

func (p *trackingBufferPool) Put(buf []byte) {
	p.released <- buf
}

// holdingBufferReader keeps the buffer passed to Read until explicitly released.
type holdingBufferReader struct {
	started chan []byte
	resume  chan struct{}
}

func (r *holdingBufferReader) Read(buf []byte) (int, error) {
	r.started <- buf
	<-r.resume
	buf[0] = 'x'
	return 1, io.EOF
}

type triggeredReader struct {
	trigger chan struct{}
	err     error
}

func (r triggeredReader) Read([]byte) (int, error) {
	<-r.trigger
	return 0, r.err
}

func TestTunnelRetainsBuffersUntilCopyFinishes(t *testing.T) {
	for _, mode := range []string{"copy error", "canceled", "deadline"} {
		for _, reverse := range []bool{false, true} {
			name := mode
			if reverse {
				name += " reversed"
			}
			t.Run(name, func(t *testing.T) {
				pool := &trackingBufferPool{
					borrowed: make(chan []byte, 2),
					released: make(chan []byte, 2),
				}
				held := &holdingBufferReader{
					started: make(chan []byte, 1),
					resume:  make(chan struct{}, 1),
				}
				defer close(held.resume)
				trigger := make(chan struct{})
				copyErr := errors.New("copy failed")
				firstErr := error(io.EOF)
				if mode == "copy error" {
					firstErr = copyErr
				}
				c1 := struct {
					io.Reader
					io.Writer
				}{triggeredReader{trigger: trigger, err: firstErr}, io.Discard}
				c2 := struct {
					io.Reader
					io.Writer
				}{held, io.Discard}
				ctx, cancel := context.WithCancel(t.Context())
				if mode == "deadline" {
					cancel()
					ctx, cancel = context.WithTimeout(t.Context(), 100*time.Millisecond)
				}
				defer cancel()
				done := make(chan error, 1)
				go func() {
					if reverse {
						done <- Tunnel(ctx, c2, c1, pool)
					} else {
						done <- Tunnel(ctx, c1, c2, pool)
					}
				}()
				var heldBuffer []byte
				select {
				case heldBuffer = <-held.started:
				case <-time.After(3 * time.Second):
					t.Fatal("copy did not start reading into its buffer")
				}
				close(trigger)
				if mode == "canceled" {
					cancel()
				}
				select {
				case err := <-done:
					want := copyErr
					if mode == "canceled" {
						want = nil
					} else if mode == "deadline" {
						want = context.DeadlineExceeded
					}
					if !errors.Is(err, want) {
						t.Fatalf("Tunnel returned %v, want %v", err, want)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("Tunnel waited for the blocked copy")
				}
				// A completed direction may release its buffer. The blocked Read
				// must retain its own buffer even after Tunnel has returned.
				returned := map[*byte]bool{}
				for range 2 {
					select {
					case buf := <-pool.released:
						if &buf[0] == &heldBuffer[0] {
							t.Fatal("buffer returned to pool while a copy still holds it")
						}
						returned[&buf[0]] = true
					default:
					}
				}
				held.resume <- struct{}{}
				for len(returned) < 2 {
					select {
					case buf := <-pool.released:
						if returned[&buf[0]] {
							t.Fatal("copy returned its buffer more than once")
						}
						returned[&buf[0]] = true
					case <-time.After(3 * time.Second):
						t.Fatal("completed copies did not return both buffers")
					}
				}
				for range 2 {
					buf := <-pool.borrowed
					if !returned[&buf[0]] {
						t.Fatal("copy did not return its borrowed buffer")
					}
				}
			})
		}
	}
}
