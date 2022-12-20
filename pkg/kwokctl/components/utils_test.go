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

package components

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestGroupByLinks(t *testing.T) {
	type args struct {
		components []internalversion.Component
	}
	tests := []struct {
		name    string
		args    args
		want    [][]internalversion.Component
		wantErr bool
	}{
		{
			name: "group by links",
			args: args{
				components: []internalversion.Component{
					{
						Name: "etcd",
					},
					{
						Name:  "kube-apiserver",
						Links: []string{"etcd"},
					},
					{
						Name:  "kube-controller-manager",
						Links: []string{"kube-apiserver"},
					},
					{
						Name:  "kube-scheduler",
						Links: []string{"kube-apiserver"},
					},
					{
						Name:  "kwok-controller",
						Links: []string{"kube-apiserver"},
					},
					{
						Name:  "prometheus",
						Links: []string{"kube-apiserver", "kube-controller-manager", "kube-scheduler", "kwok-controller"},
					},
				},
			},
			want: [][]internalversion.Component{
				{{Name: "etcd"}},
				{{Name: "kube-apiserver", Links: []string{"etcd"}}},
				{
					{Name: "kube-controller-manager", Links: []string{"kube-apiserver"}},
					{Name: "kube-scheduler", Links: []string{"kube-apiserver"}},
					{Name: "kwok-controller", Links: []string{"kube-apiserver"}},
				},
				{{Name: "prometheus", Links: []string{"kube-apiserver", "kube-controller-manager", "kube-scheduler", "kwok-controller"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GroupByLinks(tt.args.components)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupByLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("GroupByLinks() diff %s", diff)
			}
		})
	}
}
