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

package log

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/exp/slog"
	"golang.org/x/term"
)

// IsTerminal returns true if the given file descriptor is a terminal.
var IsTerminal = term.IsTerminal

// FromContext returns the Logger associated with ctx, or the default logger.
func FromContext(ctx context.Context) *Logger {
	return wrapSlog(slog.FromContext(ctx))
}

// NewContext returns a new context with the given logger.
func NewContext(ctx context.Context, logger *Logger) context.Context {
	return slog.NewContext(ctx, logger.log)
}

// NewLogger returns a new Logger that writes to w.
func NewLogger(w io.Writer, level slog.Level) *Logger {
	if w == nil {
		return noop
	}

	if file, ok := w.(*os.File); ok {
		fd := int(file.Fd())
		if IsTerminal(fd) {
			return wrapSlog(slog.New(newCtlHandler(w, fd, level)))
		}
	}

	handler := slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.AnyKind {
				if t, ok := a.Value.Any().(fmt.Stringer); ok {
					return slog.Attr{
						Key:   a.Key,
						Value: slog.StringValue(t.String()),
					}
				}
			}
			return a
		},
	}
	return wrapSlog(slog.New(handler.NewJSONHandler(w)))
}
