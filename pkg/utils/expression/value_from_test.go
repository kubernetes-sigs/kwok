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

package expression

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestDurationFrom_Get(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nowPlusOneSecond := metav1.NewTime(now.Add(time.Second))

	type args struct {
		value *time.Duration
		src   *string
		v     interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    time.Duration
		wantOk  bool
	}{
		{
			name: "No value, no src",
			args: args{
				value: nil,
				src:   nil,
				v:     corev1.Pod{},
			},
			wantOk: false,
		},
		{
			name: "Value 0, no src",
			args: args{
				value: format.Ptr(time.Duration(0)),
				src:   nil,
				v:     corev1.Pod{},
			},
			want:   0,
			wantOk: true,
		},
		{
			name: "Value 1, valid src",
			args: args{
				value: format.Ptr(time.Duration(1)),
				src:   format.Ptr(".metadata.deletionTimestamp"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &nowPlusOneSecond,
					},
				},
			},
			want:   1 * time.Second,
			wantOk: true,
		},
		{
			name: "Valid RFC3339 time src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.annotations.validRFC3339"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validRFC3339": now.Add(time.Hour).Format(time.RFC3339Nano),
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
		{
			name: "Invalid RFC3339 time src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.annotations.invalidRFC3339"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"invalidRFC3339": "invalid-time",
						},
					},
				},
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Valid duration string src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.annotations.validDuration"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validDuration": "1h",
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
		{
			name: "Empty string src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.annotations.emptyString"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"emptyString": "",
						},
					},
				},
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Noop duration",
			args: args{
				value: nil,
				src:   nil,
				v:     nil,
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Simple integer duration",
			args: args{
				value: format.Ptr(time.Duration(5)),
				src:   nil,
				v:     nil,
			},
			want:   5,
			wantOk: true,
		},
		{
			name: "Nil value with nil src",
			args: args{
				value: nil,
				src:   nil,
				v:     corev1.Pod{},
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Nil value with valid src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.deletionTimestamp"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: &nowPlusOneSecond,
					},
				},
			},
			want:   1 * time.Second,
			wantOk: true,
		},
		{
			name: "Valid duration string with nil value",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.annotations.validDuration"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validDuration": "2h",
						},
					},
				},
			},
			want:   2 * time.Hour,
			wantOk: true,
		},
		{
			name: "Non-existent field in src",
			args: args{
				value: nil,
				src:   format.Ptr(".metadata.nonExistentField"),
				v:     corev1.Pod{},
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Non-string result from query",
			args: args{
				value: nil,
				src:   format.Ptr(".spec.containers"),
				v: corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "container1"},
						},
					},
				},
			},
			want:   0,
			wantOk: false,
		},
		{
			name: "Value present with valid RFC3339 time src",
			args: args{
				value: format.Ptr(time.Duration(5)),
				src:   format.Ptr(".metadata.annotations.validRFC3339"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validRFC3339": now.Add(time.Hour).Format(time.RFC3339Nano),
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
		{
			name: "Value present with valid duration string src",
			args: args{
				value: format.Ptr(time.Duration(5)),
				src:   format.Ptr(".metadata.annotations.validDuration"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validDuration": "1h",
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
		{
			name: "Value present with non-existent field in src",
			args: args{
				value: format.Ptr(time.Duration(5)),
				src:   format.Ptr(".metadata.nonExistentField"),
				v:     corev1.Pod{},
			},
			want:   5,
			wantOk: true,
		},
		{
			name: "Value 0 with valid RFC3339 time src",
			args: args{
				value: format.Ptr(time.Duration(0)),
				src:   format.Ptr(".metadata.annotations.validRFC3339"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validRFC3339": now.Add(time.Hour).Format(time.RFC3339Nano),
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
		{
			name: "Value 0 with valid duration string src",
			args: args{
				value: format.Ptr(time.Duration(0)),
				src:   format.Ptr(".metadata.annotations.validDuration"),
				v: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"validDuration": "1h",
						},
					},
				},
			},
			want:   time.Hour,
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDurationFrom(tt.args.value, tt.args.src)
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
