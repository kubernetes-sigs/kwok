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
	"slices"
	"strings"
	"text/template"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/pools"
)

// FuncMap is a map of functions that can be used in templates.
type FuncMap = template.FuncMap

// Renderer is a template Renderer interface.
// It can render a template with the given text and original object.
type Renderer interface {
	ToText(text string, original interface{}) ([]byte, error)
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

func (r *renderer) render(buf *bytes.Buffer, text string, original interface{}) error {
	text = strings.TrimSpace(text)
	temp, ok := r.cache.Load(text)
	if !ok {
		var err error
		temp, err = template.New("_").Funcs(r.funcMap).Parse(text)
		if err != nil {
			return err
		}
		r.cache.Store(text, temp)
	}

	buf.Reset()
	err := json.NewEncoder(buf).Encode(original)
	if err != nil {
		return err
	}

	var data interface{}
	decoder := json.NewDecoder(buf)
	decoder.UseNumber()
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	buf.Reset()
	err = temp.Execute(buf, data)
	if err != nil {
		return err
	}
	return nil
}

// ToText renders the template with the given text and original object.
func (r *renderer) ToText(text string, original interface{}) ([]byte, error) {
	buf := r.bufferPool.Get()
	defer r.bufferPool.Put(buf)

	err := r.render(buf, text, original)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, buf.String())
	}
	return slices.Clone(buf.Bytes()), nil
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
