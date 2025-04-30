/*
Copyright 2025 The Kubernetes Authors.

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

package lifecycle

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func Test_newDurationFrom_Get(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowPlugOneSecond := metav1.NewTime(now.Add(time.Second))
	type args struct {
		value *time.Duration
		src   *internalversion.ExpressionFrom
		event *Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    time.Duration
		wantOk  bool
	}{
		{
			args: args{
				value: nil,
				src:   nil,
				event: &Event{
					Data: &corev1.Pod{},
				},
			},
			wantOk: false,
		},
		{
			args: args{
				value: format.Ptr(time.Duration(0)),
				src:   nil,
				event: &Event{
					Data: &corev1.Pod{},
				},
			},
			want:   0,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr(time.Duration(1)),
				src: &internalversion.ExpressionFrom{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.deletionTimestamp",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &nowPlugOneSecond,
						},
					},
				},
			},
			want:   1 * time.Second,
			wantOk: true,
		},
		{
			args: args{
				src: &internalversion.ExpressionFrom{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.annotations[\"custom-duration\"]",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"custom-duration": "7s"},
						},
					},
				},
			},
			want:   7 * time.Second,
			wantOk: true,
		},
		{
			args: args{
				src: &internalversion.ExpressionFrom{
					CEL: &internalversion.ExpressionCEL{
						Expression: "duration(self.metadata.annotations[\"custom-duration\"])",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"custom-duration": "7s"},
						},
					},
				},
			},
			want:   7 * time.Second,
			wantOk: true,
		},
		{
			args: args{
				src: &internalversion.ExpressionFrom{
					CEL: &internalversion.ExpressionCEL{
						Expression: "self.spec.duration",
					},
				},
				event: &Event{
					Data: &unstructured.Unstructured{
						Object: map[string]any{
							"spec": map[string]any{
								"duration": "7s",
							},
						},
					},
				},
			},
			want:   7 * time.Second,
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, _ := cel.NewEnvironment(cel.EnvironmentConfig{
				Types:       cel.DefaultTypes,
				Conversions: cel.DefaultConversions,
				Methods:     cel.FuncsToMethods(cel.DefaultFuncs),
				Funcs:       cel.DefaultFuncs,
				Vars:        tt.args.event.toCELStandardTypeOnly(),
			})

			d, err := newDurationFrom(tt.args.value, env, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("newDurationFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, gotOk, err := d.Get(context.Background(), tt.args.event, now)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Get() gotOk = %v, wantOk %v", gotOk, tt.wantOk)
			}
		})
	}
}
