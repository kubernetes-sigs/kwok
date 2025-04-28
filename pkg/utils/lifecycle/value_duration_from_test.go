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
package lifecycle

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestDurationFrom_Get(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowPlugOneSecond := metav1.NewTime(now.Add(time.Second))
	type args struct {
		value *time.Duration
		src   *internalversion.ExpressionFromSource
		v     *Event
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
				v: &Event{
					Data: corev1.Pod{},
				},
			},
			wantOk: false,
		},
		{
			args: args{
				value: format.Ptr(time.Duration(0)),
				src:   nil,
				v: &Event{
					Data: corev1.Pod{},
				},
			},
			want:   0,
			wantOk: true,
		},
		{
			args: args{
				value: format.Ptr(time.Duration(1)),
				src: &internalversion.ExpressionFromSource{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.deletionTimestamp",
					},
				},
				v: &Event{
					Data: corev1.Pod{
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
				src: &internalversion.ExpressionFromSource{
					JQ: &internalversion.ExpressionJQ{
						Expression: ".metadata.annotations[\"custom-duration\"]",
					},
				},
				v: &Event{
					Data: corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"custom-duration": "7s"},
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
			d, err := NewDurationFrom(tt.args.value, nil, tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDurationFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, gotOk := d.Get(context.Background(), tt.args.v, now)
			if got != tt.want {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Get() gotOk = %v, wantOk %v", gotOk, tt.wantOk)
			}
		})
	}
}
