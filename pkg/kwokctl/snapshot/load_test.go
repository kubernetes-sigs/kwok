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

package snapshot

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_load(t *testing.T) {
	controller := true
	type args struct {
		input []runtime.Object
	}
	tests := []struct {
		name        string
		args        args
		want        []runtime.Object
		wantErr     bool
		wantUpdated []runtime.Object
	}{
		{
			args: args{
				input: []runtime.Object{
					&appsv1.DaemonSet{
						TypeMeta: metav1.TypeMeta{
							Kind: "DaemonSet",
						},
						ObjectMeta: metav1.ObjectMeta{
							UID: "1",
						},
					},
					&corev1.Pod{
						TypeMeta: metav1.TypeMeta{
							Kind: "Pod",
						},
						ObjectMeta: metav1.ObjectMeta{
							UID: "2",
							OwnerReferences: []metav1.OwnerReference{
								{
									Controller: &controller,
									UID:        "1",
								},
							},
						},
					},
				},
			},
			wantUpdated: []runtime.Object{
				&appsv1.DaemonSet{
					TypeMeta: metav1.TypeMeta{
						Kind: "DaemonSet",
					},
					ObjectMeta: metav1.ObjectMeta{
						UID: "10",
					},
				},
				&corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
					ObjectMeta: metav1.ObjectMeta{
						UID: "20",
						OwnerReferences: []metav1.OwnerReference{
							{
								Controller: &controller,
								UID:        "10",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := []*unstructured.Unstructured{}
			apply := func(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
				ret := []*unstructured.Unstructured{}
				for _, obj := range objs {
					o := obj.DeepCopyObject().(*unstructured.Unstructured)
					o.SetUID(obj.GetUID() + "0")
					ret = append(ret, o)
					updated = append(updated, o)
				}
				return ret, nil
			}
			input := []*unstructured.Unstructured{}
			for _, i := range tt.args.input {
				tmp, _ := json.Marshal(i)
				u := &unstructured.Unstructured{}
				err := u.UnmarshalJSON(tmp)
				if err != nil {
					t.Errorf("failed to unmarshal: %v", err)
				}
				input = append(input, u)
			}

			got, err := load(input, apply)
			if (err != nil) != tt.wantErr {
				t.Errorf("load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			want := ObjectListToUnstructuredList(tt.want)
			if !equality.Semantic.DeepEqual(got, want) {
				t.Errorf("expected vs got:\n%s", cmp.Diff(want, got))
			}

			wantUpdated := ObjectListToUnstructuredList(tt.wantUpdated)
			if !equality.Semantic.DeepEqual(updated, wantUpdated) {
				t.Errorf("expected vs got:\n%s", cmp.Diff(updated, wantUpdated))
			}
		})
	}
}

func Test_decodeObjects(t *testing.T) {
	type args struct {
		data io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []runtime.Object
		wantErr bool
	}{
		{
			args: args{
				data: bytes.NewBufferString(`
apiVersion: v1
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
`),
			},
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
			got, err := decodeObjects(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeObjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			want := ObjectListToUnstructuredList(tt.want)
			if !equality.Semantic.DeepEqual(got, want) {
				t.Errorf("expected vs got:\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func ObjectListToUnstructuredList(objects []runtime.Object) []*unstructured.Unstructured {
	out := []*unstructured.Unstructured{}
	for _, obj := range objects {
		out = append(out, ObjectToUnstructured(obj))
	}
	return out
}

func ObjectToUnstructured(object runtime.Object) *unstructured.Unstructured {
	data, _ := json.Marshal(object)
	u := &unstructured.Unstructured{}
	_ = u.UnmarshalJSON(data)
	return u
}
