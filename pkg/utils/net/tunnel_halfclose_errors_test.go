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
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

type halfCloseErrorStream struct {
	io.ReadWriter
	closeErr error
	closed   chan struct{}
}

func (s halfCloseErrorStream) CloseWrite() error {
	close(s.closed)
	return s.closeErr
}

func TestTunnelHalfCloseErrors(t *testing.T) {
	timeoutErr := &net.OpError{Op: "read", Net: "test", Err: os.ErrDeadlineExceeded}
	closeErr := errors.New("close write failed")
	for _, tc := range []struct {
		name       string
		copyErr    error
		closeWrite bool
		closeErr   error
		reverseErr bool
		want       []error
	}{
		{
			name:       "EOF then CloseWrite error",
			copyErr:    io.EOF,
			closeWrite: true,
			closeErr:   closeErr,
			want:       []error{closeErr},
		},
		{
			name:       "timeout then CloseWrite error",
			copyErr:    timeoutErr,
			closeWrite: true,
			closeErr:   closeErr,
			want:       []error{timeoutErr, closeErr},
		},
		{
			name:    "timeout without CloseWrite",
			copyErr: timeoutErr,
			want:    []error{timeoutErr},
		},
		{
			name:       "timeout then reverse terminal error",
			copyErr:    timeoutErr,
			closeWrite: true,
			reverseErr: true,
			want:       []error{io.ErrClosedPipe},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocked := newBlockedStream()
			defer func() { _ = blocked.w.Close() }()
			c1 := struct {
				io.Reader
				io.Writer
			}{errorReader{err: tc.copyErr}, io.Discard}
			var c2 io.ReadWriter = blocked
			halfClosed := make(chan struct{})
			if tc.closeWrite {
				c2 = halfCloseErrorStream{ReadWriter: blocked, closeErr: tc.closeErr, closed: halfClosed}
			}
			pool := &trackingBufferPool{
				borrowed: make(chan []byte, 2),
				released: make(chan []byte, 2),
			}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			exited := make(chan struct{})
			go func() {
				defer close(exited)
				done <- Tunnel(ctx, c1, c2, pool)
			}()
			released := 0
			t.Cleanup(func() {
				cancel()
				blocked.unblock()
				select {
				case <-exited:
				case <-time.After(tcpTunnelTestTimeout):
					t.Error("Tunnel did not exit during cleanup")
				}
				for released < 2 {
					select {
					case <-pool.released:
						released++
					case <-time.After(tcpTunnelTestTimeout):
						t.Error("copy did not return its buffer during cleanup")
						return
					}
				}
			})
			if tc.reverseErr {
				select {
				case <-halfClosed:
				case <-time.After(tcpTunnelTestTimeout):
					t.Fatal("timeout did not half-close the destination")
				}
				// Put runs after the copy reports its result, so the timeout is
				// queued before we allow the reverse terminal error to occur.
				select {
				case <-pool.released:
					released++
				case <-time.After(tcpTunnelTestTimeout):
					t.Fatal("timed out copy did not finish")
				}
				blocked.unblock()
			}
			select {
			case err := <-done:
				for _, want := range tc.want {
					if !errors.Is(err, want) {
						t.Errorf("Tunnel returned %v, want an error matching %v", err, want)
					}
				}
			case <-time.After(tcpTunnelTestTimeout):
				t.Fatal("Tunnel waited for a blocked reverse copy after a terminal error")
			}
		})
	}
}
