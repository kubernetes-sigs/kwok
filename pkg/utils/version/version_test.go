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

package version

import (
	"reflect"
	"testing"

	"github.com/blang/semver/v4"
)

func TestParseFromOutput(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    semver.Version
		wantErr bool
	}{
		{
			args: args{
				s: "Kubernetes v1.26.0",
			},
			want: semver.MustParse("1.26.0"),
		},
		{
			args: args{
				s: "prometheus, version 2.35.0 (branch: HEAD)",
			},
			want: semver.MustParse("2.35.0"),
		},
		{
			args: args{
				s: "kwok version v0.1.0",
			},
			want: semver.MustParse("0.1.0"),
		},
		{
			args: args{
				s: "etcd Version: 3.5.6",
			},
			want: semver.MustParse("3.5.6"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFromOutput(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFromOutput() got = %v, want %v", got, tt.want)
			}
		})
	}
}
