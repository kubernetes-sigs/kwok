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

package gotpl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"

	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/pools"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// FuncMap is a map of functions that can be used in templates.
type FuncMap = template.FuncMap

// Renderer is a template Renderer interface.
// It can render a template with the given text and original object.
type Renderer interface {
	ToText(text string, original interface{}) (string, error)
	ToJSON(text string, original interface{}) ([]byte, error)
}

// renderer is a template renderer.
type renderer struct {
	cache      maps.SyncMap[string, *template.Template]
	bufferPool *pools.Pool[*bytes.Buffer]
	funcMap    template.FuncMap
}

// NewRenderer creates a new renderer.
func NewRenderer(funcMap FuncMap) Renderer {
	return &renderer{
		funcMap: funcMap,
		bufferPool: pools.NewPool(func() *bytes.Buffer {
			return bytes.NewBuffer(make([]byte, 4*1024))
		}),
	}
}

func isTemplate(temp *template.Template) bool {
	if temp == nil {
		return false
	}

	if temp.Root == nil {
		return false
	}

	if temp.Root.Type() != parse.NodeList {
		return true
	}

	if len(temp.Root.Nodes) != 1 {
		return true
	}

	if temp.Root.Nodes[0].Type() != parse.NodeText {
		return true
	}

	return false
}

func (r *renderer) render(buf *bytes.Buffer, text string, original interface{}) error {
	text = strings.TrimSpace(text)
	temp, ok := r.cache.Load(text)
	if !ok {
		var err error
		temp, err = template.New("_").
			Funcs(genericFuncs).
			Funcs(defaultFuncs).
			Funcs(r.funcMap).
			Parse(text)
		if err != nil {
			return fmt.Errorf("build template: %w", err)
		}
		if isTemplate(temp) {
			r.cache.Store(text, temp)
		} else {
			r.cache.Store(text, nil)
		}
	}

	if temp == nil {
		_, err := buf.WriteString(text)
		if err != nil {
			return fmt.Errorf("write string: %w", err)
		}
		return nil
	}

	buf.Reset()
	err := json.NewEncoder(buf).Encode(original)
	if err != nil {
		return fmt.Errorf("json encoding: %w", err)
	}

	var data interface{}
	decoder := json.NewDecoder(buf)
	decoder.UseNumber()
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("json decoding: %w", err)
	}

	buf.Reset()
	err = temp.Execute(buf, data)
	if err != nil {
		return fmt.Errorf("gotpl execute: %w", err)
	}
	return nil
}

// ToText renders the template with the given text and original object.
func (r *renderer) ToText(text string, original interface{}) (string, error) {
	buf := r.bufferPool.Get()
	defer r.bufferPool.Put(buf)

	err := r.render(buf, text, original)
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, buf.String())
	}
	return buf.String(), nil
}

// ToJSON renders the template with the given text and original object and converts the result to JSON.
func (r *renderer) ToJSON(text string, original interface{}) ([]byte, error) {
	buf := r.bufferPool.Get()
	defer r.bufferPool.Put(buf)

	err := r.render(buf, text, original)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, buf.String())
	}

	out, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, buf.String())
	}
	return out, nil
}
