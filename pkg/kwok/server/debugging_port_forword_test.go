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

func Test_findPortInForwards(t *testing.T) {
	type args struct {
		port     int32
		forwards []internalversion.Forward
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.Forward
		wantOk bool
	}{
		{
			name: "find port in forwards",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{8080},
					},
				},
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
			wantOk: true,
		},
		{
			name: "not find port in forwards",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{8081},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find port in empty forwards",
			args: args{
				port:     8080,
				forwards: []internalversion.Forward{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find port in empty ports",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{},
					},
				},
			},
			want: &internalversion.Forward{
				Ports: []int32{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findPortInForwards(tt.args.port, tt.args.forwards)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findPortInForwards() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findPortInForwards() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getPodsForward(t *testing.T) {
	type args struct {
		rules        []*internalversion.PortForward
		clusterRules []*internalversion.ClusterPortForward
		podName      string
		podNamespace string
		port         int32
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.Forward
		wantErr bool
	}{
		{
			name: "find port in rules",
			args: args{
				rules: []*internalversion.PortForward{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						Spec: internalversion.PortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				clusterRules: []*internalversion.ClusterPortForward{},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
		},
		{
			name: "find port in cluster rules",
			args: args{
				rules: []*internalversion.PortForward{},
				clusterRules: []*internalversion.ClusterPortForward{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterPortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
		},
		{
			name: "not find port in rules",
			args: args{
				rules: []*internalversion.PortForward{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2",
							Namespace: "default",
						},
						Spec: internalversion.PortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				clusterRules: []*internalversion.ClusterPortForward{},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find port in cluster rules",
			args: args{
				rules: []*internalversion.PortForward{},
				clusterRules: []*internalversion.ClusterPortForward{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterPortForwardSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPodsForward(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodsForward() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodsForward() got = %v, want %v", got, tt.want)
			}
		})
	}
}
