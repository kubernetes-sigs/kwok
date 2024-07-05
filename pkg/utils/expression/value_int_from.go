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

package expression

import (
	"context"
	"strconv"
)

// IntGetter is a interface that can be used to get a int64 value.
type IntGetter interface {
	// Get returns a int64 value.
	Get(ctx context.Context, v interface{}) (int64, bool)
}

type int64From struct {
	value *int64
	query *Query
}

// NewIntFrom returns a new IntGetter.
func NewIntFrom(value *int64, src *string) (IntGetter, error) {
	if value == nil && src == nil {
		return int64Noop{}, nil
	}
	if src == nil {
		return intWrapper(*value), nil
	}
	query, err := NewQuery(*src)
	if err != nil {
		return nil, err
	}
	return &int64From{
		value: value,
		query: query,
	}, nil
}

func (d *int64From) Get(ctx context.Context, v interface{}) (int64, bool) {
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
	switch t := out[0].(type) {
	case string:
		if t == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(t, 0, 0)
		if err == nil {
			return n, true
		}
		return 0, false
	case float64: // TODO: don't use float
		return int64(t), true
	}
	if d.value != nil {
		return *d.value, true
	}
	return 0, false
}

type int64Noop struct {
}

func (int64Noop) Get(ctx context.Context, v interface{}) (int64, bool) {
	return 0, false
}

type intWrapper int64

func (i intWrapper) Get(ctx context.Context, v interface{}) (int64, bool) {
	return int64(i), true
}
