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
	"sort"
	"testing"

	"golang.org/x/exp/slog" //nolint:depguard
)

func Test_quoteRangeTable(t *testing.T) {
	r16 := quoteRangeTable.R16
	for _, r := range r16 {
		if r.Lo > r.Hi {
			t.Errorf("quoteRangeTable has invalid range: %v", r)
		}
	}
	// This test ensures that the quoteRangeTable is sorted.
	isSorted := sort.SliceIsSorted(r16, func(i, j int) bool {
		return r16[i].Lo < r16[j].Lo
	})
	if !isSorted {
		t.Errorf("quoteRangeTable is not sorted")
	}
}

func Test_quoteIfNeed(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"empty",
			args{
				s: "",
			},
			``,
		},
		{
			"simple",
			args{
				s: "simple",
			},
			`simple`,
		},
		{
			"simple with space",
			args{
				s: "simple with space",
			},
			`"simple with space"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteIfNeed(tt.args.s); got != tt.want {
				t.Errorf("quoteIfNeed() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_formatLog(t *testing.T) {
	type args struct {
		msg       string
		attrsStr  string
		level     Level
		termWidth int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "msg",
			args: args{
				msg: "msg",
			},
			want: "msg\n",
		},
		{
			name: "msg with attrs",
			args: args{
				msg:      "msg",
				attrsStr: `a=b`,
			},
			want: "msg a=b\n",
		},
		{
			name: "msg with attrs and level",
			args: args{
				msg:      "msg",
				attrsStr: `a=b`,
				level:    LevelDebug,
			},
			want: "\x1b[0;36mDEBUG\x1b[0m msg a=b\n",
		},
		{
			name: "msg with attrs and termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				termWidth: 20,
			},
			want: "msg              a=b\n",
		},
		{
			name: "msg with attrs and level and termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				level:     LevelDebug,
				termWidth: 20,
			},
			want: "\x1b[0;36mDEBUG\x1b[0m msg        a=b\n",
		},
		{
			name: "msg with attrs and 5 termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				termWidth: 5,
			},
			want: "msg a=b\n",
		},
		{
			name: "msg with attrs and 6 termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				termWidth: 6,
			},
			want: "msg a=b\n",
		},
		{
			name: "msg with attrs and 7 termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				termWidth: 7,
			},
			want: "msg a=b\n",
		},
		{
			name: "msg with attrs and 8 termWidth",
			args: args{
				msg:       "msg",
				attrsStr:  `a=b`,
				termWidth: 8,
			},
			want: "msg  a=b\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatLog(tt.args.msg, tt.args.attrsStr, tt.args.level, tt.args.termWidth); got != tt.want {
				t.Errorf("formatLog() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
