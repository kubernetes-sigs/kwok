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

package log

import (
	"errors"
	"testing"

	"golang.org/x/exp/slog"
)

func Test_formatValue(t *testing.T) {
	type args struct {
		val slog.Value
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "format for error",
			args: args{
				val: slog.AnyValue(errors.New("unknown command \"subcommand\" for \"kwokctl\"")),
			},
			want: quoteIfNeed(errors.New("unknown command \"subcommand\" for \"kwokctl\"").Error()),
		},
		{
			name: "format for string",
			args: args{
				val: slog.AnyValue("unknown command \"subcommand\" for \"kwokctl\""),
			},
			want: quoteIfNeed("unknown command \"subcommand\" for \"kwokctl\""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.args.val); got != tt.want {
				t.Errorf("formatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
