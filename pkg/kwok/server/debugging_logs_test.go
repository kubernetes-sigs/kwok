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
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

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
						Name:      "test",
						Namespace: "default",
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
						Name: "cluster-test",
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
						Name:      "test-2",
						Namespace: "default",
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
						Name: "cluster-test",
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

func Test_tailLogsContent(t *testing.T) {
	ptr := func(i int64) *int64 { return &i }
	tests := []struct {
		name      string
		content   string
		tailLines *int64
		want      string
	}{
		{name: "no tail", content: "a\nb\nc\n", tailLines: nil, want: "a\nb\nc\n"},
		{name: "tail zero", content: "a\nb\nc\n", tailLines: ptr(0), want: ""},
		{name: "tail one", content: "a\nb\nc\n", tailLines: ptr(1), want: "c\n"},
		{name: "tail two", content: "a\nb\nc\n", tailLines: ptr(2), want: "b\nc\n"},
		{name: "tail all", content: "a\nb\nc\n", tailLines: ptr(3), want: "a\nb\nc\n"},
		{name: "tail more than lines", content: "a\nb\nc\n", tailLines: ptr(10), want: "a\nb\nc\n"},
		{name: "tail without trailing newline", content: "a\nb\nc", tailLines: ptr(1), want: "c"},
		{name: "tail empty content", content: "", tailLines: ptr(1), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tailLogsContent(tt.content, tt.tailLines); got != tt.want {
				t.Errorf("tailLogsContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_limitLogsContent(t *testing.T) {
	ptr := func(i int64) *int64 { return &i }
	tests := []struct {
		name       string
		content    string
		limitBytes *int64
		want       string
	}{
		{name: "no limit", content: "a\nb\n", limitBytes: nil, want: "a\nb\n"},
		{name: "limit zero", content: "a\nb\n", limitBytes: ptr(0), want: ""},
		{name: "limit part", content: "a\nb\n", limitBytes: ptr(3), want: "a\nb"},
		{name: "limit more than content", content: "a\nb\n", limitBytes: ptr(10), want: "a\nb\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limitLogsContent(tt.content, tt.limitBytes); got != tt.want {
				t.Errorf("limitLogsContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_readLogsContent(t *testing.T) {
	t.Run("no follow", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		err := readLogsContent(context.Background(), "a\nb\n", false, buf)
		if err != nil {
			t.Fatalf("readLogsContent() error = %v", err)
		}
		if got := buf.String(); got != "a\nb\n" {
			t.Errorf("readLogsContent() = %q, want %q", got, "a\nb\n")
		}
	})

	t.Run("follow until context done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		buf := bytes.NewBuffer(nil)
		done := make(chan error, 1)
		go func() {
			done <- readLogsContent(ctx, "a\n", true, buf)
		}()

		select {
		case <-done:
			t.Fatal("readLogsContent() returned before context done")
		case <-time.After(100 * time.Millisecond):
		}

		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("readLogsContent() error = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("readLogsContent() did not return after context done")
		}
		if got := buf.String(); got != "a\n" {
			t.Errorf("readLogsContent() = %q, want %q", got, "a\n")
		}
	})
}
