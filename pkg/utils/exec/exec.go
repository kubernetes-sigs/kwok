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

package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// IOStreams contains the standard streams.
type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

type optCtx int

// Options is the options for executing a command.
type Options struct {
	// Dir is the working directory of the command.
	Dir string
	// Env is the environment variables of the command.
	Env []string
	// UID is the user id of the command
	UID *int64
	// GID is the group id of the command
	GID *int64
	// IOStreams contains the standard streams.
	IOStreams
	// PipeStdin is true if the command's stdin should be piped.
	PipeStdin bool
	// Fork is true if the command should be forked.
	Fork bool
}

func (e *Options) deepCopy() *Options {
	return &Options{
		Dir:       e.Dir,
		Env:       append([]string(nil), e.Env...),
		GID:       e.GID,
		UID:       e.UID,
		IOStreams: e.IOStreams,
		PipeStdin: e.PipeStdin,
		Fork:      e.Fork,
	}
}

// WithPipeStdin returns a context with the given pipeStdin option.
func WithPipeStdin(ctx context.Context, pipeStdin bool) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.PipeStdin = pipeStdin
	return ctx
}

// WithEnv returns a context with the given environment variables.
func WithEnv(ctx context.Context, env []string) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.Env = append(opt.Env, env...)
	return ctx
}

// WithUser returns a context with the given username and group name.
func WithUser(ctx context.Context, uid, gid *int64) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.UID = uid
	opt.GID = gid
	return ctx
}

// WithDir returns a context with the given working directory.
func WithDir(ctx context.Context, dir string) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.Dir = dir
	return ctx
}

// WithIOStreams returns a context with the given IOStreams.
func WithIOStreams(ctx context.Context, streams IOStreams) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.IOStreams = streams
	return ctx
}

// WithStdIO returns a context with the standard IOStreams.
func WithStdIO(ctx context.Context) context.Context {
	return WithIOStreams(ctx, IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})
}

// WithReadWriter returns a context with the given io.ReadWriter as the In and Out streams.
func WithReadWriter(ctx context.Context, rw io.ReadWriter) context.Context {
	return WithIOStreams(ctx, IOStreams{
		In:  rw,
		Out: rw,
	})
}

// WithAllWriteTo returns a context with the given io.Writer as the In, Out, and ErrOut streams.
func WithAllWriteTo(ctx context.Context, w io.Writer) context.Context {
	return WithIOStreams(ctx, IOStreams{
		ErrOut: w,
		Out:    w,
	})
}

// WithWriteTo returns a context with the given io.Writer as the Out stream.
func WithWriteTo(ctx context.Context, w io.Writer) context.Context {
	return WithIOStreams(ctx, IOStreams{
		Out: w,
	})
}

// WithAllWriteToErrOut returns a context with the given io.Writer as the ErrOut stream.
func WithAllWriteToErrOut(ctx context.Context) context.Context {
	return WithAllWriteTo(ctx, os.Stderr)
}

// WithFork returns a context with the given fork option.
func WithFork(ctx context.Context, fork bool) context.Context {
	ctx, opt := withExecOptions(ctx)
	opt.Fork = fork
	return ctx
}

func withExecOptions(ctx context.Context) (context.Context, *Options) {
	v := ctx.Value(optCtx(0))
	if v == nil {
		opt := &Options{}
		return context.WithValue(ctx, optCtx(0), opt), opt
	}
	opt := v.(*Options).deepCopy()
	return context.WithValue(ctx, optCtx(0), opt), opt
}

// GetExecOptions returns the ExecOptions for the given context.
func GetExecOptions(ctx context.Context) *Options {
	v := ctx.Value(optCtx(0))
	if v == nil {
		return &Options{}
	}
	return v.(*Options).deepCopy()
}

// Exec executes the given command and returns the output.
func Exec(ctx context.Context, name string, args ...string) error {
	_, err := Command(ctx, name, args...)
	return err
}

// Command executes the given command and return the command.
func Command(ctx context.Context, name string, args ...string) (cmd *exec.Cmd, err error) {
	opt := GetExecOptions(ctx)
	if opt.Fork {
		subCtx := context.Background()
		cmd = startProcess(subCtx, name, args...)
	} else {
		cmd = command(ctx, name, args...)
	}
	if opt.Env != nil {
		cmd.Env = append(os.Environ(), opt.Env...)
	}
	if err = setUser(cmd, opt.UID, opt.GID); err != nil {
		return nil, fmt.Errorf("cmd set user: %s %s: %w", name, strings.Join(args, " "), err)
	}

	cmd.Dir = opt.Dir

	if opt.In != nil {
		if opt.PipeStdin {
			inPipe, err := cmd.StdinPipe()
			if err != nil {
				return nil, err
			}
			go func() {
				_, _ = io.Copy(inPipe, opt.In)
			}()
		} else {
			cmd.Stdin = opt.In
		}
	}

	cmd.Stdout = opt.Out
	cmd.Stderr = opt.ErrOut

	if !opt.Fork && cmd.Stderr == nil {
		buf := bytes.NewBuffer(nil)
		cmd.Stderr = buf
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cmd start: %s %s: %w", name, strings.Join(args, " "), err)
	}

	if !opt.Fork {
		err = cmd.Wait()
		if err != nil {
			if buf, ok := cmd.Stderr.(*bytes.Buffer); ok {
				return nil, fmt.Errorf("cmd wait: %s %s: %w\n%s", name, strings.Join(args, " "), err, buf.String())
			}
			return nil, fmt.Errorf("cmd wait: %s %s: %w", name, strings.Join(args, " "), err)
		}
	}
	return cmd, nil
}
