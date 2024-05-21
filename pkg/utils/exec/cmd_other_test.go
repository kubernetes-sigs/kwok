//go:build !windows

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

package exec

import (
	"context"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
)

func Test_startProcess(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
		arg  []string
	}
	tests := []struct {
		name string
		args args
		want *exec.Cmd
	}{
		{
			name: "Test with valid arguments",
			args: args{
				ctx:  context.Background(),
				name: "echo",
				arg:  []string{"ec"},
			},
			want: exec.Command("echo", "ec"),
		},
		{
			name: "TestStartProcessWithEcho",
			args: args{
				ctx:  context.Background(),
				name: "echo",
				arg:  []string{"hello"},
			},
			want: exec.Command("echo", "hello"),
		},
		{
			name: "TestStartProcessWithDir",
			args: args{
				ctx:  context.Background(),
				name: "ls",
				arg:  []string{"-l"},
			},
			want: exec.Command("ls", "-l"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := startProcess(tt.args.ctx, tt.args.name, tt.args.arg...); !reflect.DeepEqual(got.Args, tt.want.Args) {
				t.Errorf("startProcess() = want %v got %v", tt.want.Args, got.Args)
			}
		})
	}
}

func Test_command(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
		arg  []string
	}
	tests := []struct {
		name string
		args args
		want *exec.Cmd
	}{
		{
			name: "Test with valid arguments",
			args: args{
				ctx:  context.Background(),
				name: "echo",
				arg:  []string{"ec"},
			},
			want: exec.Command("echo", "ec"),
		},
		{
			name: "TestStartProcessWithEcho",
			args: args{
				ctx:  context.Background(),
				name: "echo",
				arg:  []string{"hello"},
			},
			want: exec.Command("echo", "hello"),
		},
		{
			name: "TestStartProcessWithDir",
			args: args{
				ctx:  context.Background(),
				name: "ls",
				arg:  []string{"-l"},
			},
			want: exec.Command("ls", "-l"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := command(tt.args.ctx, tt.args.name, tt.args.arg...); !reflect.DeepEqual(got.Args, tt.want.Args) {
				t.Errorf("command() = want %v got %v", tt.want.Args, got.Args)
			}
		})
	}
}

func Test_isRunning(t *testing.T) {
	type args struct {
		pid int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Running",
			args: args{
				1,
			},
			want: true,
		},
		{
			name: "NotRunning",
			args: args{
				0,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRunning(tt.args.pid); got != tt.want {
				t.Errorf("isRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetUser(t *testing.T) {
	// Mock cmd
	cmd := &exec.Cmd{
		SysProcAttr: &syscall.SysProcAttr{},
	}

	// Test cases
	tests := []struct {
		name string
		args args
		want *syscall.Credential
	}{
		{
			name: "Both UID and GID provided",
			args: args{uid: intPtr(1000), gid: intPtr(1000)},
			want: &syscall.Credential{Uid: 1000, Gid: 1000},
		},
		{
			name: "Only UID provided",
			args: args{uid: intPtr(1000), gid: nil},
			want: &syscall.Credential{Uid: 1000, Gid: 1000}, // Assuming GID is retrieved correctly
		},

		{
			name: "No UID or GID provided",
			args: args{uid: nil, gid: nil},
			want: nil,
		},
		{
			name: "Negative UID and GID provided",
			args: args{uid: intPtr(-1), gid: intPtr(-1)},
			want: nil, // Error condition
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setUser(cmd, tt.args.uid, tt.args.gid)
			if err != nil && tt.want == nil {
				// If an error is expected and occurred, test passed
				return
			}
			if err != nil && tt.want != nil {
				// If an error is expected but did not occur, test failed
				t.Errorf("setUser() error = %v, want nil", err)
				return
			}
			if tt.want == nil {
				// Both UID and GID are nil, no further checks needed
				return
			}
			if cmd.SysProcAttr == nil || cmd.SysProcAttr.Credential == nil {
				t.Errorf("setUser() did not set SysProcAttr.Credential, want %v", tt.want)
				return
			}
			if cmd.SysProcAttr.Credential.Uid != tt.want.Uid || cmd.SysProcAttr.Credential.Gid != tt.want.Gid {
				t.Errorf("setUser() did not set correct UID or GID, got %v, want %v", cmd.SysProcAttr.Credential, tt.want)
			}
		})
	}
}

// Helper function to convert int64 to *int64
func intPtr(n int64) *int64 {
	return &n
}

type args struct {
	uid *int64
	gid *int64
}
