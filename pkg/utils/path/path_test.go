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

package path

import (
	"os"
	"runtime"
	"testing"
)

func TestHome(t *testing.T) {
	home := Home()

	if home == "" {
		t.Errorf("Expected home directory, but got empty string")
	}

	if home[0] != '/' {
		t.Errorf("Expected home directory to be absolute path, but got %s", home)
	}

	if home[len(home)-1] == '/' {
		t.Errorf("Expected home directory to be absolute path, but got %s", home)
	}
}

func TestExpand(t *testing.T) {
	home := Home()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	var testCases = []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "tilde and dot",
			input:    "~/./example.txt",
			expected: Join(home, "example.txt"),
			wantErr:  false,
		},
		{
			name:     "tilde and dotdot",
			input:    "~/../example.txt",
			expected: Join(home, "../example.txt"),
			wantErr:  false,
		},
		{
			name:     "tilde",
			input:    "~",
			expected: home,
			wantErr:  false,
		},
		{
			name:     "tilde slash",
			input:    "~/example.txt",
			expected: Join(home, "example.txt"),
			wantErr:  false,
		},
		{
			name:     "absolute path",
			input:    "/example.txt",
			expected: "/example.txt",
			wantErr:  false,
		},
		{
			name:     "pwd path",
			input:    "example.txt",
			expected: Join(wd, "example.txt"),
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := Expand(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if output != tc.expected {
				t.Errorf("Expected %s, but got %s", tc.expected, output)
			}
		})
	}
}

func TestRelFromHome(t *testing.T) {
	type args struct {
		target string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				target: "",
			},
			want: "",
		},
		{
			name: "home",
			args: args{
				target: Home(),
			},
			want: "~",
		},
		{
			name: "home slash",
			args: args{
				target: Home() + "/",
			},
			want: "~",
		},
		{
			name: "home slash file",
			args: args{
				target: Home() + "/file",
			},
			want: "~/file",
		},
		{
			name: "home slash file slash",
			args: args{
				target: Home() + "/file/",
			},
			want: "~/file",
		},
		{
			name: "out of home",
			args: args{
				target: "/tmp",
			},
			want: "/tmp",
		},
		{
			name: "out of home slash",
			args: args{
				target: "/tmp/",
			},
			want: "/tmp/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RelFromHome(tt.args.target); got != tt.want {
				t.Errorf("RelFromHome() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	type args struct {
		elem []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				elem: []string{},
			},
			want: ".",
		},
		{
			name: "empty string",
			args: args{
				elem: []string{""},
			},
			want: ".",
		},
		{
			name: "slash",
			args: args{
				elem: []string{"/"},
			},
			want: "/",
		},
		{
			name: "slash slash",
			args: args{
				elem: []string{"/", "/"},
			},
			want: "/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Join(tt.args.elem...); got != tt.want {
				t.Errorf("Join() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClean(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{
				p: "",
			},
			want: ".",
		},
		{
			name: "backslash",
			args: args{
				p: "\\",
			},
			want: func() string {
				if runtime.GOOS == "windows" {
					return "//"
				}
				return "\\"
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Clean(tt.args.p); got != tt.want {
				t.Errorf("Clean() = %v, want %v", got, tt.want)
			}
		})
	}
}
