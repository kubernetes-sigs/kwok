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
	"encoding/json"
	"fmt"
)

// NewParameters parses the parameters.
func NewParameters(ctx context.Context, raw json.RawMessage, params []string) (any, error) {
	var param any
	err := json.Unmarshal(raw, &param)
	if err != nil {
		return nil, fmt.Errorf("unmarshal params error: %w", err)
	}

	for _, p := range params {
		q, err := NewQuery(p)
		if err != nil {
			return nil, fmt.Errorf("parse param %s error: %w", p, err)
		}
		datas, err := q.Execute(ctx, param)
		if err != nil {
			return nil, fmt.Errorf("execute param %s with %v error: %w", p, param, err)
		}
		if len(datas) != 1 {
			return nil, fmt.Errorf("unexpected result: %v", datas)
		}
		param = datas[0]
	}

	return param, nil
}
