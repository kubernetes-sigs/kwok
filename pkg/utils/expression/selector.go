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
	"strconv"

	"k8s.io/utils/strings/slices"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

// Requirement contains values, a key, and an operator that relates the key and values.
// The zero value of Requirement is invalid.
type Requirement struct {
	query     *Query
	operator  internalversion.SelectorOperator
	strValues []string
}

// NewRequirement is the constructor for a Requirement.
func NewRequirement(key string, op internalversion.SelectorOperator, vals []string) (*Requirement, error) {
	q, err := NewQuery(key)
	if err != nil {
		return nil, err
	}

	switch op {
	case internalversion.SelectorOpIn, internalversion.SelectorOpNotIn:
		if len(vals) == 0 {
			return nil, fmt.Errorf("for 'in', 'notin' operators, values set can't be empty")
		}
	case internalversion.SelectorOpExists, internalversion.SelectorOpDoesNotExist:
		if len(vals) != 0 {
			return nil, fmt.Errorf("values set must be empty for exists and does not exist")
		}
	default:
		return nil, fmt.Errorf("operator %q is not supported", op)
	}

	return &Requirement{query: q, operator: op, strValues: vals}, nil
}

// Matches returns true if the Requirement matches the input Labels.
// There is a match in the following cases:
func (r *Requirement) Matches(ctx context.Context, matchData interface{}) (bool, error) {
	data, err := r.query.Execute(ctx, matchData)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, nil
	}
	switch r.operator {
	case internalversion.SelectorOpIn:
		return hasValues(data, r.strValues), nil
	case internalversion.SelectorOpNotIn:
		return !hasValues(data, r.strValues), nil
	case internalversion.SelectorOpExists:
		return existsValue(data), nil
	case internalversion.SelectorOpDoesNotExist:
		return !existsValue(data), nil
	default:
		return false, nil
	}
}

func hasValues(v []interface{}, vs []string) bool {
	for _, d := range v {
		if hasValue(d, vs) {
			return true
		}
	}
	return false
}

func hasValue(d interface{}, vs []string) bool {
	switch t := d.(type) {
	case string:
		return slices.Contains(vs, t)
	case bool:
		return slices.Contains(vs, strconv.FormatBool(t))
	case int:
		return slices.Contains(vs, strconv.FormatInt(int64(t), 10))
	}
	return false
}

func existsValue(vs []interface{}) bool {
	for _, d := range vs {
		if d != nil {
			return true
		}
	}
	return false
}
