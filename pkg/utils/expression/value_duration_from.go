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

package expression

import (
	"context"
	"fmt"
	"time"
)

// DurationGetter is a interface that can be used to get a time.Duration value.
type DurationGetter interface {
	// Get returns a duration value.
	Get(ctx context.Context, v interface{}, now time.Time) (time.Duration, bool)
	// Info return a duration information
	Info(ctx context.Context, v interface{}) (string, bool)
}

type durationFrom struct {
	value *time.Duration
	query *Query
}

// NewDurationFrom returns a new DurationGetter.
func NewDurationFrom(value *time.Duration, src *string) (DurationGetter, error) {
	if value == nil && src == nil {
		return durationNoop{}, nil
	}
	if src == nil {
		return duration(*value), nil
	}
	query, err := NewQuery(*src)
	if err != nil {
		return nil, err
	}
	return &durationFrom{
		value: value,
		query: query,
	}, nil
}

func (d *durationFrom) Get(ctx context.Context, v interface{}, now time.Time) (time.Duration, bool) {
	out, err := d.query.Execute(ctx, v)
	if err != nil {
		return 0, false
	}
	if len(out) == 0 {
		if d.value != nil {
			return *d.value, true
		}
		return 0, false
	}
	if t, ok := out[0].(string); ok {
		if t == "" {
			return 0, false
		}
		ti, err := time.Parse(time.RFC3339Nano, t)
		if err == nil {
			d := ti.Sub(now)
			return d, true
		}
		du, err := time.ParseDuration(t)
		if err == nil {
			return du, true
		}
	}
	return 0, false
}

func (d *durationFrom) Info(ctx context.Context, v interface{}) (string, bool) {
	out, err := d.query.Execute(ctx, v)
	if err != nil {
		return "", false
	}
	if len(out) == 0 {
		if d.value != nil {
			return d.value.String(), true
		}
		return "", false
	}
	if t, ok := out[0].(string); ok {
		if t == "" {
			return "", false
		}
		_, err := time.Parse(time.RFC3339Nano, t)
		if err == nil {
			return fmt.Sprintf("(now - %q)", t), true
		}
		du, err := time.ParseDuration(t)
		if err == nil {
			return du.String(), true
		}
	}
	return "", false
}

type durationNoop struct {
}

func (durationNoop) Get(ctx context.Context, v interface{}, now time.Time) (time.Duration, bool) {
	return 0, false
}

func (durationNoop) Info(ctx context.Context, v interface{}) (string, bool) {
	return "", false
}

type duration int64

func (i duration) Get(ctx context.Context, v interface{}, now time.Time) (time.Duration, bool) {
	return time.Duration(i), true
}

func (i duration) Info(ctx context.Context, v interface{}) (string, bool) {
	return time.Duration(i).String(), false
}
