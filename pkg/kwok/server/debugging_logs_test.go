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

func Test_findLogInLogs(t *testing.T) {
	type args struct {
		containerName string
		logs          []internalversion.Log
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.Log
		wantOk bool
	}{
		{
			name: "find log in logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{"test"},
					},
				},
			},
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
			wantOk: true,
		},
		{
			name: "not find log in logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{"test1"},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find log in empty logs",
			args: args{
				containerName: "test",
				logs:          []internalversion.Log{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find log in empty logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{},
					},
				},
			},
			want: &internalversion.Log{
				Containers: []string{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findLogInLogs(tt.args.containerName, tt.args.logs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findLogInLogs() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findLogInLogs() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getPodLogs(t *testing.T) {
	type args struct {
		rules         []*internalversion.Logs
		clusterRules  []*internalversion.ClusterLogs
		podName       string
		podNamespace  string
		containerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.Log
		wantErr bool
	}{
		{
			name: "find logs in rule",
			args: args{
				rules: []*internalversion.Logs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						Spec: internalversion.LogsSpec{
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterLogs{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
		},
		{
			name: "find logs in cluster rule",
			args: args{
				rules: []*internalversion.Logs{},
				clusterRules: []*internalversion.ClusterLogs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterLogsSpec{
							Logs: []internalversion.Log{
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
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
		},
		{
			name: "not find logs in rule",
			args: args{
				rules: []*internalversion.Logs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2",
							Namespace: "default",
						},
						Spec: internalversion.LogsSpec{
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterLogs{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find logs in cluster rule",
			args: args{
				rules: []*internalversion.Logs{},
				clusterRules: []*internalversion.ClusterLogs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterLogsSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Logs: []internalversion.Log{
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
			got, err := getPodLogs(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodLogs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
