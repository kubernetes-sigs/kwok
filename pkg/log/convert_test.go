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

package log

import "testing"

func TestToLogSeverityLevel(t *testing.T) {
	type args struct {
		level Level
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "debuglevel,-1",
			args: args{
				level: -1,
			},
			want: DebugLevelSecurity,
		},
		{
			name: "debuglevel, -100",
			args: args{
				level: -100,
			},
			want: DebugLevelSecurity,
		},
		{
			name: "infolevel, 1",
			args: args{
				level: 1,
			},
			want: InfoLevelSecurity,
		},
		{
			name: "infolevel, 3",
			args: args{
				level: 3,
			},
			want: InfoLevelSecurity,
		},
		{
			name: "warnlevel, 5",
			args: args{
				level: 5,
			},
			want: WarnLevelSecurity,
		},
		{
			name: "warnlevel, 7",
			args: args{
				level: 7,
			},
			want: WarnLevelSecurity,
		},
		{
			name: "errorlevel, 8",
			args: args{
				level: 8,
			},
			want: ErrorLevelSecurity,
		},
		{
			name: "errorlevel, 100",
			args: args{
				level: 100,
			},
			want: ErrorLevelSecurity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToLogSeverityLevel(tt.args.level); got != tt.want {
				t.Errorf("ToLogSeverityLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
