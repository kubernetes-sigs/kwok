//go:build !windows

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

package exec

import (
	"os"
	"os/exec"
	"syscall"
	"testing"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

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
				os.Getpid(),
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

	type args struct {
		uid *int64
		gid *int64
	}

	// Test cases
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Both UID and GID provided",
			args: args{uid: format.Ptr(int64(os.Getuid())), gid: format.Ptr(int64(os.Getgid()))},
		},
		{
			name: "Only UID provided",
			args: args{uid: format.Ptr(int64(os.Getuid())), gid: nil},
		},

		{
			name: "No UID or GID provided",
			args: args{uid: nil, gid: nil},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setUser(cmd, tt.args.uid, tt.args.gid)
			if err != nil {
				t.Errorf("setUser() error = %v, want nil", err)
				return
			}

		})
	}
}
