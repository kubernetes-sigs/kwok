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

package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"sigs.k8s.io/kwok/pkg/log"
)

func ForkExec(ctx context.Context, dir string, name string, arg ...string) error {
	pidPath := PathJoin(dir, "pids", filepath.Base(name)+".pid")
	pidData, err := os.ReadFile(pidPath)
	if err == nil {
		pid, err := strconv.Atoi(string(pidData))
		if err == nil {
			if isRunning(pid) {
				return nil
			}
		}
	}

	logPath := PathJoin(dir, "logs", filepath.Base(name)+".log")
	cmdlinePath := PathJoin(dir, "cmdline", filepath.Base(name))

	err = os.MkdirAll(filepath.Dir(pidPath), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(cmdlinePath), 0755)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	args := append([]string{name}, arg...)

	err = os.WriteFile(cmdlinePath, []byte(strings.Join(args, "\x00")), 0644)
	if err != nil {
		return fmt.Errorf("write cmdline file %s: %w", cmdlinePath, err)
	}

	cmd := startProcess(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	err = cmd.Start()
	if err != nil {
		return err
	}

	err = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	if err != nil {
		return fmt.Errorf("write pid file %s: %w", pidPath, err)
	}
	return nil
}

func ForkExecRestart(ctx context.Context, dir string, name string) error {
	cmdlinePath := PathJoin(dir, "cmdline", filepath.Base(name))

	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return err
	}

	args := strings.Split(string(data), "\x00")

	return ForkExec(ctx, dir, args[0], args[1:]...)
}

func ForkExecKill(ctx context.Context, dir string, name string) error {
	pidPath := PathJoin(dir, "pids", filepath.Base(name)+".pid")
	_, err := os.Stat(pidPath)
	if err != nil {
		// No pid file exists, which means the process has been terminated
		logger := log.FromContext(ctx)
		logger.Debug("Stat file",
			"path", pidPath,
			"err", err,
		)
		return nil
	}
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("read pid file %s: %w", pidPath, err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil {
		return fmt.Errorf("parse pid file %s: %w", pidPath, err)
	}
	err = killProcess(pid)
	if err != nil {
		return err
	}
	return os.Remove(pidPath)
}

func Exec(ctx context.Context, dir string, stm IOStreams, name string, arg ...string) error {
	cmd := command(ctx, name, arg...)
	cmd.Dir = dir
	cmd.Stdin = stm.In
	cmd.Stdout = stm.Out
	cmd.Stderr = stm.ErrOut

	if cmd.Stderr == nil {
		buf := bytes.NewBuffer(nil)
		cmd.Stderr = buf
	}
	err := cmd.Run()
	if err != nil {
		if buf, ok := cmd.Stderr.(*bytes.Buffer); ok {
			return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(arg, " "), err, buf.String())
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(arg, " "), err)
	}
	return nil
}

type IOStreams struct {
	// In think, os.Stdin
	In io.Reader
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	err = process.Kill()
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return fmt.Errorf("kill process: %w", err)
	}
	_, err = process.Wait()
	if err != nil {
		if errors.Is(err, syscall.ECHILD) {
			return nil
		}
		return err
	}
	return nil
}
