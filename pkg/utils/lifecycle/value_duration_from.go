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

package lifecycle

import (
	"context"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/expression"
)

// durationGetter is a interface that can be used to get a time.Duration value.
type durationGetter interface {
	// Get returns a duration value.
	Get(ctx context.Context, event *Event, now time.Time) (time.Duration, bool)
}

type durationFrom struct {
	value *time.Duration
	query *expression.Query
	cel   cel.Program
}

// NewDurationFrom returns a new DurationGetter.
func NewDurationFrom(value *time.Duration, env *cel.Environment, src *internalversion.ExpressionFromSource) (durationGetter, error) {
	if value == nil && src == nil {
		return durationNoop{}, nil
	}
	if src == nil {
		return duration(*value), nil
	}

	d := &durationFrom{
		value: value,
	}

	if src.CEL != nil {
		program, err := env.Compile(src.CEL.Expression)
		if err != nil {
			return nil, err
		}
		d.cel = program
	}

	if src.JQ != nil {
		query, err := expression.NewQuery(src.JQ.Expression)
		if err != nil {
			return nil, err
		}
		d.query = query
	}
	return d, nil
}

func (d *durationFrom) Get(ctx context.Context, event *Event, now time.Time) (time.Duration, bool) {
	if d.cel != nil {
		out, _, err := d.cel.Eval(event.Data)
		if err != nil {
			return d.defaultValue()
		}

		n, err := cel.AsDuration(out)
		if err != nil {
			return d.defaultValue()
		}

		return n, true
	}

	if d.query != nil {
		out, err := d.query.Execute(ctx, event.toJSONStandard())
		if err != nil {
			return 0, false
		}
		if len(out) == 0 {
			return d.defaultValue()
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
	}
	return d.defaultValue()
}

func (d *durationFrom) defaultValue() (time.Duration, bool) {
	if d.value != nil {
		return *d.value, true
	}
	return 0, false
}

type durationNoop struct {
}

func (durationNoop) Get(ctx context.Context, event *Event, now time.Time) (time.Duration, bool) {
	return 0, false
}

type duration int64

func (i duration) Get(ctx context.Context, event *Event, now time.Time) (time.Duration, bool) {
	return time.Duration(i), true
}
