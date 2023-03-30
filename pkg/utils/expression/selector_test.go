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

package expression

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestRequirement_Matches(t *testing.T) {
	type src struct {
		key  string
		op   internalversion.SelectorOperator
		vals []string
	}
	type args struct {
		matchData interface{}
	}
	tests := []struct {
		name    string
		src     src
		args    args
		wantErr bool
		wantOk  bool
	}{
		{
			src: src{
				key: ".status.containerStatuses.[].state.waiting.reason",
				op:  "In",
				vals: []string{
					"ContainerCreating",
					"Failed",
				},
			},
			args: args{
				&corev1.Pod{
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
			wantOk:  true,
			wantErr: false,
		},
		{
			src: src{
				key: ".status.podIP",
				op:  "NotIn",
				vals: []string{
					"ContainerCreating",
					"Failed",
				},
			},
			args: args{
				&corev1.Pod{}},
			wantOk:  true,
			wantErr: false,
		},
		{
			src: src{
				key:  ".status.conditions.[] | select( .reason == \"PodScheduled\" ) | .status",
				op:   "Exists",
				vals: []string{},
			},
			args: args{
				&corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Reason: "PodScheduled",
								Status: "True",
							},
						},
					},
				}},
			wantOk:  true,
			wantErr: false,
		},
		{
			src: src{
				key:  ".status.conditions.[] | select( .reason == \"PodScheduled\" ) | .status",
				op:   "DoesNotExist",
				vals: []string{},
			},
			args: args{
				&corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Reason: "PodScheduled",
								Status: "True",
							},
						},
					},
				}},
			wantOk:  false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewRequirement(tt.src.key, tt.src.op, tt.src.vals)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRequirement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotOk, err := d.Matches(context.Background(), tt.args.matchData)
			if (err != nil) != tt.wantErr {
				t.Errorf("Matches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOk != tt.wantOk {
				t.Errorf("Matches() gotOk = %v, wantOk %v", gotOk, tt.wantOk)
			}
		})
	}
}
