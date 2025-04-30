/*
Copyright 2025 The Kubernetes Authors.

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

// int64Getter is a interface that can be used to get a int64 value.
type int64Getter interface {
	// Get returns a int64 value.
	Get(ctx context.Context, event *Event) (int64, bool, error)
}

type int64From struct {
	value   *int64
	query   *expression.Query
	program cel.Program
}

// newInt64From returns a new int64Getter.
func newInt64From(value *int64, env *cel.Environment, src *internalversion.ExpressionFrom) (int64Getter, error) {
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
		d.program = program
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

func (d *int64From) Get(ctx context.Context, event *Event) (int64, bool, error) {
	if d.program != nil {
		out, _, err := d.program.Eval(event.toCELStandard())
		if err != nil {
			return 0, false, err
		}

		n, err := cel.AsInt64(out)
		if err != nil {
			return 0, false, err
		}

		return n, true, nil
	}

	if d.query != nil {
		out, err := d.query.Execute(ctx, event.toJSONStandard())
		if err != nil {
			return 0, false, err
		}
		if len(out) == 0 {
			return d.defaultValue()
		}
		switch t := out[0].(type) {
		case string:
			if t == "" {
				return 0, false, nil
			}
			n, err := strconv.ParseInt(t, 0, 0)
			if err == nil {
				return n, true, nil
			}
			return 0, false, nil
		case float64: // TODO: don't use float
			return int64(t), true, nil
		}
	}

	return d.defaultValue()
}

func (d *int64From) defaultValue() (int64, bool, error) {
	if d.value != nil {
		return *d.value, true, nil
	}
	return 0, false, nil
}

type int64Noop struct {
}

func (int64Noop) Get(ctx context.Context, event *Event) (int64, bool, error) {
	return 0, false, nil
}

type intWrapper int64

func (i intWrapper) Get(ctx context.Context, event *Event) (int64, bool, error) {
	return int64(i), true, nil
}
