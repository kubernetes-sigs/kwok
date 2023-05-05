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

	//nolint:depguard
	"golang.org/x/exp/slog"
)

var noop = wrapSlog(noopHandler{}, slog.LevelInfo)

type noopHandler struct{}

var _ slog.Handler = noopHandler{}

func (noopHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (noopHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h noopHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h noopHandler) WithGroup(name string) slog.Handler {
	return h
}
