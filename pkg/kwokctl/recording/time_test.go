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

package recording

import (
	"reflect"
	"testing"
	"time"
)

func TestReplaceTimeToRelative(t *testing.T) {
	type args struct {
		baseTime time.Time
		data     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "replace time to relative",
			args: args{
				baseTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				data:     "any xxx 2021-01-01T00:00:01Z any xxx",
			},
			want: "any xxx $(time-offset-nanosecond 1000000000) any xxx",
		},
		{
			name: "replace time to relative with nanosecond",
			args: args{
				baseTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				data:     "any xxx 2021-01-01T00:00:01.1Z any xxx",
			},
			want: "any xxx $(time-offset-nanosecond 1100000000) any xxx",
		},
		{
			name: "replace time to relative with nanosecond",
			args: args{
				baseTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				data:     "any xxx 2021-01-01T00:00:01.00000Z any xxx",
			},
			want: "any xxx $(time-offset-nanosecond 1000000000) any xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(ReplaceTimeToRelative(tt.args.baseTime, []byte(tt.args.data))); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReplaceTimeToRelative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRevertTimeFromRelative(t *testing.T) {
	type args struct {
		baseTime time.Time
		data     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "revert time from relative",
			args: args{
				baseTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				data:     "any xxx $(time-offset-nanosecond 1000000000) any xxx",
			},
			want: "any xxx 2021-01-01T00:00:01.000000Z any xxx",
		},
		{
			name: "revert time from relative with nanosecond",
			args: args{
				baseTime: time.Date(2021, 1, 1, 0, 0, 0, 1, time.UTC),
				data:     "any xxx $(time-offset-nanosecond 1100000000) any xxx",
			},
			want: "any xxx 2021-01-01T00:00:01.100000Z any xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(RevertTimeFromRelative(tt.args.baseTime, []byte(tt.args.data))); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RevertTimeFromRelative() = %v, want %v", got, tt.want)
			}
		})
	}
}
