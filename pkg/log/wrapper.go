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
	"fmt"
	"strconv"
	"strings"

	//nolint:depguard
	"golang.org/x/exp/slog"
)

// Level is the logging level.
type Level = slog.Level

// The following is Level definitions copied from slog.
const (
	DebugLevel Level = slog.DebugLevel
	InfoLevel  Level = slog.InfoLevel
	WarnLevel  Level = slog.WarnLevel
	ErrorLevel Level = slog.ErrorLevel
)

func wrapSlog(log *slog.Logger) *Logger {
	return &Logger{log}
}

// Logger is a wrapper around slog.Logger.
type Logger struct {
	log *slog.Logger
}

// LogDepth logs a message with the given level and depth.
func (l *Logger) LogDepth(calldepth int, level Level, msg string, args ...any) {
	l.log.LogDepth(calldepth+1, level, msg, args...)
}

// Log logs a message with the given level.
func (l *Logger) Log(level Level, msg string, args ...any) {
	l.LogDepth(0, level, msg, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.LogDepth(0, DebugLevel, msg, args...)
}

// Info logs an informational message.
func (l *Logger) Info(msg string, args ...any) {
	l.LogDepth(0, InfoLevel, msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.LogDepth(0, WarnLevel, msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args[:len(args):len(args)], slog.Any(slog.ErrorKey, err))
	}
	l.log.LogDepth(0, ErrorLevel, msg, args...)
}

// With returns a new Logger that includes the given arguments.
func (l *Logger) With(args ...any) *Logger {
	return wrapSlog(l.log.With(args...))
}

// WithGroup returns a new Logger that starts a group. The keys of all
// attributes added to the Logger will be qualified by the given name.
func (l *Logger) WithGroup(name string) *Logger {
	return wrapSlog(l.log.WithGroup(name))
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
		l = DebugLevel
	case "INFO":
		l = InfoLevel
	case "WARN":
		l = WarnLevel
	case "ERROR":
		l = ErrorLevel
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
