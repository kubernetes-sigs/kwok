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

package config

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/kwok/pkg/log"
)

type configCtx int

type configValue struct {
	Objects     []InternalObject
	Unsupported []json.RawMessage
}

// setupContext sets the given objects in the context.
func setupContext(ctx context.Context, objs []InternalObject, unsupported []json.RawMessage) context.Context {
	val := &configValue{
		Objects:     objs,
		Unsupported: unsupported,
	}
	return context.WithValue(ctx, configCtx(0), val)
}

// addToContext adds the given objects to the context.
func addToContext(ctx context.Context, objs ...InternalObject) {
	v := ctx.Value(configCtx(0))
	val, ok := v.(*configValue)
	if !ok {
		logger := log.FromContext(ctx)
		logger.Warn("Unable to add to context")
		return
	}

	val.Objects = append(val.Objects, objs...)
}

// GetFromContext returns the objects from the context.
func GetFromContext(ctx context.Context) []InternalObject {
	v := ctx.Value(configCtx(0))
	val, ok := v.(*configValue)
	if !ok {
		logger := log.FromContext(ctx)
		logger.Warn("Unable to get from context")
		return nil
	}

	return val.Objects
}

// GetUnsupportedFromContext returns the unsupported objects from the context.
func GetUnsupportedFromContext(ctx context.Context) []json.RawMessage {
	v := ctx.Value(configCtx(0))
	val, ok := v.(*configValue)
	if !ok {
		logger := log.FromContext(ctx)
		logger.Warn("Unable to get unsupported from context")
		return nil
	}

	return val.Unsupported
}
