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

package controllers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestNewStagesFromYaml(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []*internalversion.Stage
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				data: []byte(`kind: Stage
apiVersion: kwok.x-k8s.io/v1alpha1
metadata:
  name: node-test
spec:
  resourceRef:
    apiGroup: v1
    kind: Node`),
			},
			want: []*internalversion.Stage{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node-test",
					},
					Spec: internalversion.StageSpec{
						ResourceRef: internalversion.StageResourceRef{
							APIGroup: "v1",
							Kind:     "Node",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStagesFromYaml(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStagesFromYaml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("NewStagesFromYaml() diff (-got +want): %s", diff)
			}
		})
	}
}
