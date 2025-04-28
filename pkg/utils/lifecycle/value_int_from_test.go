/*
Copyright 2024 The Kubernetes Authors.

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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestNewIntFrom_Get(t *testing.T) {
	type args struct {
		value *int64
		src   *internalversion.ExpressionFromSource
		v     *Event
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
				v: &Event{
					Data: corev1.Pod{},
				},
			},
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src:   nil,
				v: &Event{
					Data: corev1.Pod{},
				},
			},
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr[int64](0),
				src: &internalversion.ExpressionFromSource{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.deletionGracePeriodSeconds",
					},
				},
				v: &Event{
					Data: corev1.Pod{
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
				src: &internalversion.ExpressionFromSource{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.generation",
					},
				},
				v: &Event{
					Data: corev1.Pod{
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
				src: &internalversion.ExpressionFromSource{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.annotations.x",
					},
				},
				v: &Event{
					Data: corev1.Pod{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewIntFrom(tt.args.value, nil, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDurationFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, got1 := d.Get(context.Background(), tt.args.v)
			if got != tt.want {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.wantOk {
				t.Errorf("Get() gotOk = %v, want %v", got1, tt.wantOk)
			}
		})
	}
}
