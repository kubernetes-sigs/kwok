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
	"runtime"
	"strconv"
	"strings"
	"time"

	//nolint:depguard
	"golang.org/x/exp/slog"
)

// Level is the logging level.
type Level = slog.Level

// The following is Level definitions copied from slog.
const (
	LevelDebug Level = slog.LevelDebug
	LevelInfo  Level = slog.LevelInfo
	LevelWarn  Level = slog.LevelWarn
	LevelError Level = slog.LevelError
)

func wrapSlog(handler slog.Handler, level slog.Level) *Logger {
	return &Logger{handler, level}
}

// Logger is a wrapper around slog.Handler.
type Logger struct {
	handler slog.Handler
	level   Level // Level specifies a level of verbosity for V logs.
}

// Enabled returns whether the given level is enabled.
func (l *Logger) Enabled(ctx context.Context, level Level) bool {
	return l.handler.Enabled(ctx, level)
}

// Log logs a message with the given level.
func (l *Logger) Log(level Level, msg string, args ...any) {
	l.log(level, msg, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an informational message.
func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args[:len(args):len(args)], slog.Any("err", err))
	}
	l.log(LevelError, msg, args...)
}

// log is the low-level logging method for methods that take ...any.
// It must always be called directly by an exported logging method
// or function, because it uses a fixed call depth to obtain the pc.
// copied from slog.Logger
func (l *Logger) log(level Level, msg string, args ...any) {
	if !l.handler.Enabled(context.Background(), level) {
		return
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	pc = pcs[0]
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.Add(args...)
	_ = l.handler.Handle(context.Background(), r)
}

// With returns a new Logger that includes the given arguments.
func (l *Logger) With(args ...any) *Logger {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return wrapSlog(l.handler.WithAttrs(attrs), l.level)
}

// WithGroup returns a new Logger that starts a group. The keys of all
// attributes added to the Logger will be qualified by the given name.
func (l *Logger) WithGroup(name string) *Logger {
	return wrapSlog(l.handler.WithGroup(name), l.level)
}

// Level returns
func (l *Logger) Level() Level {
	return l.level
}

// ParseLevel parses a level string.
func ParseLevel(s string) (l Level, err error) {
	name := s
	offsetStr := ""
	i := strings.IndexAny(s, "+-")
	if i > 0 {
		name = s[:i]
		offsetStr = s[i:]
	} else if i == 0 ||
		(name[0] >= '0' && name[0] <= '9') {
		name = "INFO"
		offsetStr = s
	}

	switch strings.ToUpper(name) {
	case "DEBUG":
		l = LevelDebug
	case "INFO":
		l = LevelInfo
	case "WARN":
		l = LevelWarn
	case "ERROR":
		l = LevelError
	default:
		return 0, fmt.Errorf("ParseLevel %q: invalid level name", s)
	}

	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, fmt.Errorf("ParseLevel %q: invalid offset: %w", s, err)
		}
		l += Level(offset)
	}

	return l, nil
}

const badKey = "!BADKEY"

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
// copied from slog.Logger
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
