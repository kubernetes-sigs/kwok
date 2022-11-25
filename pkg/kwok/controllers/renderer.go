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

package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"sigs.k8s.io/yaml"
)

type renderer struct {
	cache      sync.Map
	bufferPool sync.Pool
	funcMap    template.FuncMap
}

func newRenderer(funcMap template.FuncMap) *renderer {
	return &renderer{
		funcMap: funcMap,
		cache:   sync.Map{},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 4*1024))
			},
		},
	}
}

// renderToJson renders the template with the given text and original object.
func (r *renderer) renderToJson(text string, original interface{}) ([]byte, error) {
	text = strings.TrimSpace(text)
	v, ok := r.cache.Load(text)
	if !ok {
		temp, err := template.New("_").Funcs(r.funcMap).Parse(text)
		if err != nil {
			return nil, err
		}
		r.cache.Store(text, temp)
		v = temp
	}
	temp := v.(*template.Template)
	buf := r.bufferPool.Get().(*bytes.Buffer)
	defer r.bufferPool.Put(buf)

	buf.Reset()
	err := json.NewEncoder(buf).Encode(original)
	if err != nil {
		return nil, err
	}

	var data interface{}
	decoder := json.NewDecoder(buf)
	decoder.UseNumber()
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	buf.Reset()
	err = temp.Execute(buf, data)
	if err != nil {
		return nil, err
	}

	out, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, buf.String())
	}
	return out, nil
}
