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
	"testing"
	"text/template"
)

func TestRenderToJson(t *testing.T) {
	testCases := []struct {
		name      string
		funcMap   template.FuncMap
		templText string
		original  interface{}
		expected  string
	}{
		{
			name:      "basic",
			funcMap:   template.FuncMap{},
			original:  map[string]interface{}{"k": "v1"},
			templText: `{"k":{{ .k }}}`,
			expected:  `{"k":"v1"}`,
		},
		{
			name:      "basic with yaml format",
			funcMap:   template.FuncMap{},
			original:  map[string]interface{}{"k": "v1"},
			templText: `k: {{ .k }}`,
			expected:  `{"k":"v1"}`,
		},
		{
			name: "with funcMap",
			funcMap: template.FuncMap{
				"Foo": func() string {
					return "foo"
				},
			},
			original:  map[string]interface{}{"k": "v1"},
			templText: `{"foo":{{ Foo }},"k":{{ .k }}}`,
			expected:  `{"foo":"foo","k":"v1"}`,
		},
		{
			name: "with whitespace",
			funcMap: template.FuncMap{
				"Foo": func() string {
					return "foo"
				},
			},
			original:  map[string]interface{}{"k": "v1"},
			templText: `        {"foo":{{ Foo }},"k":{{ .k }}}       `,
			expected:  `{"foo":"foo","k":"v1"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRenderer(tc.funcMap)
			actual, err := r.renderToJson(tc.templText, tc.original)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, actual)
			}
		})
	}
}
