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

package progressbar

import (
	"testing"
)

func Test_formatInfo(t *testing.T) {
	type args struct {
		max      uint64
		preInfo  string
		postInfo string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				max:      17,
				preInfo:  "preInfo",
				postInfo: "postInfo",
			},
			want: "preInfo  postInfo",
		},
		{
			args: args{
				max:      16,
				preInfo:  "preInfo",
				postInfo: "postInfo",
			},
			want: "preInfo postInfo",
		},
		{
			args: args{
				max:      15,
				preInfo:  "preInfo",
				postInfo: "postInfo",
			},
			want: "pr...o postInfo",
		},
		{
			args: args{
				max:      14,
				preInfo:  "preInfo",
				postInfo: "postInfo",
			},
			want: "p...o postInfo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatInfo(tt.args.max, tt.args.preInfo, tt.args.postInfo); got != tt.want {
				t.Errorf("formatInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
