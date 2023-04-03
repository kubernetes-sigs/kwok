//go:build !windows

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

package server

import (
	"context"
	"io"

	cpty "github.com/creack/pty"
	clientremotecommand "k8s.io/client-go/tools/remotecommand"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
)

func (s *Server) execInContainerWithTTY(ctx context.Context, cmd []string, in io.Reader, out io.WriteCloser, resize <-chan clientremotecommand.TerminalSize) error {
	logger := log.FromContext(ctx)

	// Create a pty.
	pty, tty, err := cpty.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = pty.Close()
		_ = tty.Close()
	}()

	// Create a two way tunnel for pty and stream.
	go func() {
		buf1 := s.bufPool.Get()
		buf2 := s.bufPool.Get()
		defer func() {
			s.bufPool.Put(buf1)
			s.bufPool.Put(buf2)
		}()
		stm := struct {
			io.Reader
			io.Writer
		}{in, out}
		err := tunnel(ctx, pty, stm, buf1, buf2)
		if err != nil {
			logger.Error("failed to tunnel", err)
		}
	}()

	// Resize pty.
	if resize != nil {
		go func() {
			for size := range resize {
				err := cpty.Setsize(pty, &cpty.Winsize{
					Rows: size.Height,
					Cols: size.Width,
				})
				if err != nil {
					logger.Error("failed to resize pty", err)
				}
			}
		}()
	}

	// Set the stream as the stdin/stdout/stderr.
	ctx = exec.WithIOStreams(ctx, exec.IOStreams{
		In:     tty,
		Out:    tty,
		ErrOut: tty,
	})

	// Execute the command.
	err = exec.Exec(ctx, cmd[0], cmd[1:]...)
	if err != nil {
		return err
	}

	return nil
}
