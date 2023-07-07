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
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

func startProcess(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := command(ctx, name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setsid is used to detach the process from the parent (normally a shell)
		Setsid: true,
	}
	return cmd
}

func command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd
}

func isRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func setUser(cmd *exec.Cmd, uid, gid *int64) error {
	if uid == nil && gid == nil {
		return nil
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	if cmd.SysProcAttr.Credential == nil {
		cmd.SysProcAttr.Credential = &syscall.Credential{}
	}
	// If both uid and gid are both set, use them directly
	if uid != nil && gid != nil {
		_, err := user.LookupId(strconv.Itoa(int(*uid)))
		if err != nil {
			return err
		}
		_, err = user.LookupGroupId(strconv.Itoa(int(*gid)))
		if err != nil {
			return err
		}
		cmd.SysProcAttr.Credential.Uid = uint32(*uid)
		cmd.SysProcAttr.Credential.Gid = uint32(*gid)
		return nil
	}
	// If only uid is set, use that user's gid
	if uid != nil {
		userInfo, err := user.LookupId(strconv.Itoa(int(*uid)))
		if err != nil {
			return err
		}
		u, err := strconv.Atoi(userInfo.Uid)
		if err != nil {
			return err
		}
		g, err := strconv.Atoi(userInfo.Gid)
		if err != nil {
			return err
		}
		cmd.SysProcAttr.Credential.Uid = uint32(u)
		cmd.SysProcAttr.Credential.Gid = uint32(g)
	}
	// If only gid is set, use the current user's uid
	if gid != nil {
		userInfo, err := user.Current()
		if err != nil {
			return err
		}
		u, err := strconv.Atoi(userInfo.Uid)
		if err != nil {
			return err
		}
		cmd.SysProcAttr.Credential.Uid = uint32(u)
		cmd.SysProcAttr.Credential.Gid = uint32(*gid)
	}
	return nil
}
