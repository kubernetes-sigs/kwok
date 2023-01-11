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

type Query struct {
	jq *gojq.Query
}

func NewQuery(src string) (*Query, error) {
	q, err := gojq.Parse(src)
	if err != nil {
		return nil, err
	}
	return &Query{
		jq: q,
	}, nil
}

func (q *Query) Execute(ctx context.Context, v interface{}) ([]interface{}, error) {
	v, err := ToJsonStandard(v)
	if err != nil {
		return nil, err
	}
	out := []interface{}{}
	iter := q.jq.RunWithContext(ctx, v)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if _, ok := v.(error); ok {
			return nil, nil
		}
		out = append(out, v)
	}
	return out, nil
}

func ToJsonStandard(v interface{}) (interface{}, error) {
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
