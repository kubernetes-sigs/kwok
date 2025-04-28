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

package lifecycle

import (
	"context"
	"strconv"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/expression"
)

// intGetter is a interface that can be used to get a int64 value.
type intGetter interface {
	// Get returns a int64 value.
	Get(ctx context.Context, event *Event) (int64, bool)
}

type int64From struct {
	value *int64
	query *expression.Query
	cel   cel.Program
}

// NewIntFrom returns a new IntGetter.
func NewIntFrom(value *int64, env *cel.Environment, src *internalversion.ExpressionFromSource) (intGetter, error) {
	if value == nil && src == nil {
		return int64Noop{}, nil
	}
	if src == nil {
		return intWrapper(*value), nil
	}

	d := &int64From{
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

func (d *int64From) Get(ctx context.Context, event *Event) (int64, bool) {
	if d.cel != nil {
		out, _, err := d.cel.Eval(event.Data)
		if err != nil {
			return d.defaultValue()
		}

		n, err := cel.AsInt(out)
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
	}

	return d.defaultValue()
}

func (d *int64From) defaultValue() (int64, bool) {
	if d.value != nil {
		return *d.value, true
	}
	return 0, false
}

type int64Noop struct {
}

func (int64Noop) Get(ctx context.Context, event *Event) (int64, bool) {
	return 0, false
}

type intWrapper int64

func (i intWrapper) Get(ctx context.Context, event *Event) (int64, bool) {
	return int64(i), true
}
