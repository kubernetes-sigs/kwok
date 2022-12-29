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

package expression

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestQuery_Execute(t *testing.T) {
	type args struct {
		src string
		v   interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			args: args{
				src: ".status.podIP",
				v:   &corev1.Pod{},
			},
			want: []interface{}{
				nil,
			},
		},
		{
			args: args{
				src: ".status.containerStatuses.[].state.waiting.reason",
				v: &corev1.Pod{
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{
							{
								State: corev1.ContainerState{
									Waiting: &corev1.ContainerStateWaiting{
										Reason: "ContainerCreating",
									},
								},
							},
							{
								State: corev1.ContainerState{
									Waiting: &corev1.ContainerStateWaiting{
										Reason: "Failed",
									},
								},
							},
						},
					},
				},
			},
			want: []interface{}{
				"ContainerCreating",
				"Failed",
			},
		},
		{
			args: args{
				src: ".status.conditions.[] | select( .reason == \"PodScheduled\" ) | .status",
				v: &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Reason: "PodScheduled",
								Status: "True",
							},
						},
					},
				},
			},
			want: []interface{}{
				"True",
			},
		},
		{
			args: args{
				src: ".metadata.finalizers.[]",
				v: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{
							"test",
						},
					},
				},
			},
			want: []interface{}{
				"test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := NewQuery(tt.args.src)
			if err != nil {
				t.Fatal(err)
			}
			got, err := q.Execute(context.Background(), tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Execute() got = %v, want %v", got, tt.want)
			}
		})
	}
}
