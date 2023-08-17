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
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/kwok/pkg/utils/slices"
)

func TestDecodeToUnstructured(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    []runtime.Object
		wantErr bool
	}{
		{
			data: `
apiVersion: v1
kind: Pod
metadata:
  name: test
  namespace: test
spec:
  containers: []
---
---
apiVersion: v1
kind: Pod
metadata:
  name: test-2
  namespace: test
spec:
  containers: []
`,
			want: []runtime.Object{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := []*unstructured.Unstructured{}
			err := NewDecoder(bytes.NewBufferString(tt.data)).DecodeToUnstructured(func(obj *unstructured.Unstructured) error {
				got = append(got, obj)
				return nil
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeToUnstructured() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			want := objectListToUnstructuredList(tt.want)
			if !equality.Semantic.DeepEqual(got, want) {
				t.Errorf("expected vs got:\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func objectListToUnstructuredList(objects []runtime.Object) []*unstructured.Unstructured {
	return slices.Map(objects, objectToUnstructured)
}

func objectToUnstructured(object runtime.Object) *unstructured.Unstructured {
	data, _ := json.Marshal(object)
	u := &unstructured.Unstructured{}
	_ = u.UnmarshalJSON(data)
	return u
}
