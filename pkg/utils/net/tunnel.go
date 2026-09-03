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

// Tunnel create tunnels for two streams.
func Tunnel(ctx context.Context, c1, c2 io.ReadWriter, buf1, buf2 []byte) error {
	// Buffered so that both senders can complete even when this function
	// returns without receiving their results, which the two ctx.Done() paths
	// below do. On an unbuffered channel those goroutines block on the send
	// forever once the caller closes the streams.
	errCh := make(chan error, 2)
	go func() {
		_, err := io.CopyBuffer(c2, c1, buf1)
		errCh <- err
	}()
	go func() {
		_, err := io.CopyBuffer(c1, c2, buf2)
		errCh <- err
	}()
	select {
	case <-ctx.Done():
		// Do nothing
	case err1 := <-errCh:
		select {
		case <-ctx.Done():
			if err1 != nil {
				return err1
			}
			// Do nothing
		case err2 := <-errCh:
			if err1 != nil {
				return err1
			}
			return err2
		}
	}
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
