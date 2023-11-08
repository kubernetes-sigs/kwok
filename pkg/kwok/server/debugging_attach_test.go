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

package server

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func Test_findAttachInAttaches(t *testing.T) {
	type args struct {
		containerName string
		attaches      []internalversion.AttachConfig
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.AttachConfig
		wantOk bool
	}{
		{
			name: "find attach in attaches",
			args: args{
				containerName: "test",
				attaches: []internalversion.AttachConfig{
					{
						Containers: []string{"test"},
					},
				},
			},
			want: &internalversion.AttachConfig{
				Containers: []string{"test"},
			},
			wantOk: true,
		},
		{
			name: "not find attach in attaches",
			args: args{
				containerName: "test",
				attaches: []internalversion.AttachConfig{
					{
						Containers: []string{"test1"},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find attach in empty attaches",
			args: args{
				containerName: "test",
				attaches:      []internalversion.AttachConfig{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find attach in empty attaches",
			args: args{
				containerName: "test",
				attaches: []internalversion.AttachConfig{
					{
						Containers: []string{},
					},
				},
			},
			want: &internalversion.AttachConfig{
				Containers: []string{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findAttachInAttaches(tt.args.containerName, tt.args.attaches)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findAttachInAttaches() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findAttachInAttaches() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getPodAttaches(t *testing.T) {
	type args struct {
		rules         []*internalversion.Attach
		clusterRules  []*internalversion.ClusterAttach
		podName       string
		podNamespace  string
		containerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.AttachConfig
		wantErr bool
	}{
		{
			name: "find attaches in rule",
			args: args{
				rules: []*internalversion.Attach{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						Spec: internalversion.AttachSpec{
							Attaches: []internalversion.AttachConfig{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterAttach{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.AttachConfig{
				Containers: []string{"test"},
			},
		},
		{
			name: "find attaches in cluster rule",
			args: args{
				rules: []*internalversion.Attach{},
				clusterRules: []*internalversion.ClusterAttach{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterAttachSpec{
							Attaches: []internalversion.AttachConfig{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.AttachConfig{
				Containers: []string{"test"},
			},
		},
		{
			name: "not find attaches in rule",
			args: args{
				rules: []*internalversion.Attach{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2",
							Namespace: "default",
						},
						Spec: internalversion.AttachSpec{
							Attaches: []internalversion.AttachConfig{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterAttach{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find attaches in cluster rule",
			args: args{
				rules: []*internalversion.Attach{},
				clusterRules: []*internalversion.ClusterAttach{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterAttachSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Attaches: []internalversion.AttachConfig{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPodAttach(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodAttaches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodAttaches() got = %v, want %v", got, tt.want)
			}
		})
	}
}
