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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/cel"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func Test_newInt64From_Get(t *testing.T) {
	type args struct {
		value *int64
		src   *internalversion.ExpressionFrom
		event *Event
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantOk  bool
		wantErr bool
	}{
		{
			args: args{
				value: nil,
				src:   nil,
				event: &Event{
					Data: &corev1.Pod{},
				},
			},
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src:   nil,
				event: &Event{
					Data: &corev1.Pod{},
				},
			},
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFrom{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.deletionGracePeriodSeconds",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							DeletionGracePeriodSeconds: format.Ptr[int64](1),
						},
					},
				},
			},
			want:   1,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFrom{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.generation",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Generation: 1,
						},
					},
				},
			},
			want:   1,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFrom{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.annotations.x",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"x": "1",
							},
						},
					},
				},
			},
			want:   1,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFrom{
					CEL: &internalversion.ExpressionCEL{
						Expression: "self.spec.terminationGracePeriodSeconds",
					},
				},
				event: &Event{
					Data: &corev1.Pod{
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: format.Ptr[int64](1),
						},
					},
				},
			},
			want:   1,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFrom{
					CEL: &internalversion.ExpressionCEL{
						Expression: "self.spec.number",
					},
				},
				event: &Event{
					Data: &unstructured.Unstructured{
						Object: map[string]any{
							"spec": map[string]any{
								"number": 10,
							},
						},
					},
				},
			},
			want:   10,
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

			d, err := newInt64From(tt.args.value, env, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("newInt64From() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, got1, err := d.Get(context.Background(), tt.args.event)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.wantOk {
				t.Errorf("Get() gotOk = %v, want %v", got1, tt.wantOk)
			}
		})
	}
}
