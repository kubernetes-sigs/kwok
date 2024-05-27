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
	"encoding/json"
	"math/big"

	"github.com/itchyny/gojq"
)

// Query is wrapper of gojq.Query.
type Query struct {
	code *gojq.Code
}

// NewQuery returns a new Query.
func NewQuery(src string) (*Query, error) {
	q, err := gojq.Parse(src)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(q)
	if err != nil {
		return nil, err
	}
	return &Query{
		code: code,
	}, nil
}

// Execute executes the query with the given value.
func (q *Query) Execute(ctx context.Context, v interface{}) ([]interface{}, error) {
	v, err := ToJSONStandard(v)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	iter := q.code.RunWithContext(ctx, v)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if _, ok := v.(error); ok {
			return nil, nil
		}
		if v == nil {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// ToJSONStandard converts the given value to a value that used by gojq.
func ToJSONStandard(v interface{}) (interface{}, error) {
	switch v.(type) {
	case nil, bool, int, float64, *big.Int, string, []interface{}, map[string]interface{}:
		return v, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var out interface{}
		err = json.Unmarshal(data, &out)
		if err != nil {
			return nil, err
		}
		return out, nil
	}
}
