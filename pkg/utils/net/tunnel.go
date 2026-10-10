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

package net

import (
	"context"
	"errors"
	"io"
)

// BufferPool provides reusable buffers for tunnel copies.
// Its methods must be safe to call concurrently.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

// Tunnel creates tunnels for two streams. A nil pool uses io.CopyBuffer's default
// buffer. Each copy retains its buffer until it finishes, even if Tunnel returns
// early. Callers must close the streams to unblock pending copies after return.
func Tunnel(ctx context.Context, c1, c2 io.ReadWriter, pool BufferPool) error {
	// Buffered so that both senders can complete even when this function
	// returns without receiving their results. On an unbuffered channel those
	// goroutines block on the send forever once the caller closes the streams.
	errCh := make(chan error, 2)
	copyStream := func(dst io.Writer, src io.Reader) {
		var buf []byte
		if pool != nil {
			buf = pool.Get()
			defer pool.Put(buf)
		}
		_, err := io.CopyBuffer(dst, src, buf)
		errCh <- err
	}
	go copyStream(c2, c1)
	go copyStream(c1, c2)
	select {
	case <-ctx.Done():
		// Do nothing
	case err1 := <-errCh:
		if err1 != nil {
			return err1
		}
		select {
		case <-ctx.Done():
			// Do nothing
		case err2 := <-errCh:
			return err2
		}
	}
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
