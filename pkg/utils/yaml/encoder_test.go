/*
Copyright 2023 The Kubernetes Authors.

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

package yaml

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEncoder(t *testing.T) {
	tests := []struct {
		name    string
		data    []runtime.Object
		want    string
		wantErr bool
	}{
		{
			data: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test",
							"namespace": "test",
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{},
						},
					},
				},
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "v1",
						"kind":       "Pod",
						"metadata": map[string]interface{}{
							"name":      "test-2",
							"namespace": "test",
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{},
						},
					},
				},
			},
			want: `apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: test
spec:
  containers: []
---
apiVersion: v1
kind: Pod
metadata:
  name: test-2
  namespace: test
spec:
  containers: []
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			encoder := NewEncoder(buf)
			for _, obj := range tt.data {
				err := encoder.Encode(obj)
				if (err != nil) != tt.wantErr {
					t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			got := buf.String()
			if !equality.Semantic.DeepEqual(tt.want, got) {
				t.Errorf("expected vs got:\n%s", cmp.Diff(tt.want, got))
				fmt.Println(got)
			}
		})
	}
}
