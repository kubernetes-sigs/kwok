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
	"strings"
	"testing"
	"time"

	"github.com/wzshiming/ctc"
)

func Test_formatProgress(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			name    string
			width   uint64
			current uint64
			total   uint64
			elapsed time.Duration
		}
		expected string
	}{
		{
			name: "Progress_with_completion",
			args: struct {
				name    string
				width   uint64
				current uint64
				total   uint64
				elapsed time.Duration
			}{
				name:    "Task",
				width:   50,
				current: 100,
				total:   100,
				elapsed: time.Second * 10,
			},
			expected: "\x1b[0mTask                                      size=100B speed=10B/s elapsed=10s\x1b[0m",
		},
		{
			name: "Progress_with_ongoing_task",
			args: struct {
				name    string
				width   uint64
				current uint64
				total   uint64
				elapsed time.Duration
			}{
				name:    "Task",
				width:   50,
				current: 50,
				total:   100,
				elapsed: time.Second * 10,
			},
			expected: "\x1b[0mTask                                      size=50B/100B speed=5B/s elapsed=10s\x1b[0m",
		},
		{
			name: "Progress_with_narrow_width",
			args: struct {
				name    string
				width   uint64
				current uint64
				total   uint64
				elapsed time.Duration
			}{
				name:    "Task",
				width:   20,
				current: 50,
				total:   100,
				elapsed: time.Second * 10,
			},
			expected: "\x1b[0mTask        size=50B/100B\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProgress(tt.args.name, tt.args.width, tt.args.current, tt.args.total, tt.args.elapsed)
			if strings.Contains(got, tt.expected) {
				t.Errorf("formatProgress() = %q, want %q", got, tt.expected)
			}
		})
	}
}
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

func Test_calculateSplitIndex(t *testing.T) {
	tests := []struct {
		name     string
		info     string
		width    uint64
		current  uint64
		total    uint64
		expected uint64
	}{
		{
			name:     "Half progress",
			info:     "progress bar",
			width:    20,
			current:  50,
			total:    100,
			expected: 10,
		},
		{
			name:     "Full progress",
			info:     "progress bar",
			width:    20,
			current:  100,
			total:    100,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSplitIndex(tt.info, tt.width, tt.current, tt.total)
			if got != tt.expected {
				t.Errorf("calculateSplitIndex() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatBar(t *testing.T) {
	tests := []struct {
		name     string
		info     string
		index    uint64
		expected string
	}{
		{
			name:     "Standard bar",
			info:     "progress bar",
			index:    8,
			expected: ctc.Reset.String() + ctc.Negative.String() + "progress" + ctc.Reset.String() + " bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBar(tt.info, tt.index)
			if got != tt.expected {
				t.Errorf("formatBar() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatSpeed(t *testing.T) {
	tests := []struct {
		name     string
		size     uint64
		elapsed  time.Duration
		expected string
	}{
		{
			name:     "Standard speed",
			size:     100,
			elapsed:  time.Second * 10,
			expected: "10B/s",
		},
		{
			name:     "Immediate speed",
			size:     100,
			elapsed:  time.Millisecond,
			expected: "100B/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSpeed(tt.size, tt.elapsed)
			if got != tt.expected {
				t.Errorf("formatSpeed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func Test_formatBytes(t *testing.T) {
	tests := []struct {
		name     string
		size     uint64
		expected string
	}{
		{
			name:     "Bytes",
			size:     100,
			expected: "100B",
		},
		{
			name:     "Kilobytes",
			size:     2048,
			expected: "2KiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.size)
			if got != tt.expected {
				t.Errorf("formatBytes() = %v, want %v", got, tt.expected)
			}
		})
	}
}
