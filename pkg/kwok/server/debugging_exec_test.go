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

func Test_findContainerInExecs(t *testing.T) {
	type args struct {
		containerName string
		execs         []internalversion.ExecTarget
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.ExecTarget
		wantOk bool
	}{
		{
			name: "find exec in execs",
			args: args{
				containerName: "test",
				execs: []internalversion.ExecTarget{
					{
						Containers: []string{"test"},
					},
				},
			},
			want: &internalversion.ExecTarget{
				Containers: []string{"test"},
			},
			wantOk: true,
		},
		{
			name: "not find exec in execs",
			args: args{
				containerName: "test",
				execs: []internalversion.ExecTarget{
					{
						Containers: []string{"test1"},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find exec in empty execs",
			args: args{
				containerName: "test",
				execs:         []internalversion.ExecTarget{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find exec in empty execs",
			args: args{
				containerName: "test",
				execs: []internalversion.ExecTarget{
					{
						Containers: []string{},
					},
				},
			},
			want: &internalversion.ExecTarget{
				Containers: []string{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findContainerInExecs(tt.args.containerName, tt.args.execs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findContainerInExecs() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findContainerInExecs() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getExecTarget(t *testing.T) {
	type args struct {
		rules         []*internalversion.Exec
		clusterRules  []*internalversion.ClusterExec
		podName       string
		podNamespace  string
		containerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.ExecTarget
		wantErr bool
	}{
		{
			name: "find execs in rule",
			args: args{
				rules: []*internalversion.Exec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						Spec: internalversion.ExecSpec{
							Execs: []internalversion.ExecTarget{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterExec{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.ExecTarget{
				Containers: []string{"test"},
			},
		},
		{
			name: "find execs in cluster rule",
			args: args{
				rules: []*internalversion.Exec{},
				clusterRules: []*internalversion.ClusterExec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterExecSpec{
							Execs: []internalversion.ExecTarget{
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
			want: &internalversion.ExecTarget{
				Containers: []string{"test"},
			},
		},
		{
			name: "not find execs in rule",
			args: args{
				rules: []*internalversion.Exec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2",
							Namespace: "default",
						},
						Spec: internalversion.ExecSpec{
							Execs: []internalversion.ExecTarget{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterExec{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find execs in cluster rule",
			args: args{
				rules: []*internalversion.Exec{},
				clusterRules: []*internalversion.ClusterExec{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterExecSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Execs: []internalversion.ExecTarget{
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
			got, err := getExecTarget(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getExecTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExecTarget() got = %v, want %v", got, tt.want)
			}
		})
	}
}
