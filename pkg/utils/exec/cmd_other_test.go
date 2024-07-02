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

	"github.com/stretchr/testify/assert"

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
	type args struct {
		uid *int64
		gid *int64
	}

	// Test cases
	tests := []struct {
		name           string
		args           args
		expectedUID    *uint32
		expectedGID    *uint32
		expectedErrMsg string
	}{
		{
			name:        "Both UID and GID provided",
			args:        args{uid: format.Ptr(int64(os.Getuid())), gid: format.Ptr(int64(os.Getgid()))},
			expectedUID: format.Ptr(uint32(os.Getuid())),
			expectedGID: format.Ptr(uint32(os.Getgid())),
		},
		{
			name:        "Only UID provided",
			args:        args{uid: format.Ptr(int64(os.Getuid())), gid: nil},
			expectedUID: format.Ptr(uint32(os.Getuid())),
			expectedGID: nil,
		},
		{
			name:        "Only GID provided",
			args:        args{uid: nil, gid: format.Ptr(int64(os.Getgid()))},
			expectedUID: nil,
			expectedGID: format.Ptr(uint32(os.Getgid())),
		},
		{
			name:        "No UID or GID provided",
			args:        args{uid: nil, gid: nil},
			expectedUID: nil,
			expectedGID: nil,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &exec.Cmd{
				SysProcAttr: &syscall.SysProcAttr{},
			}

			err := setUser(cmd, tt.args.uid, tt.args.gid)

			if tt.expectedErrMsg != "" {
				assert.EqualError(t, err, tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedUID != nil {
				assert.NotNil(t, cmd.SysProcAttr.Credential)
				assert.Equal(t, *tt.expectedUID, cmd.SysProcAttr.Credential.Uid)
			}

			if tt.expectedGID != nil {
				assert.NotNil(t, cmd.SysProcAttr.Credential)
				assert.Equal(t, *tt.expectedGID, cmd.SysProcAttr.Credential.Gid)
			}
		})
	}
}
