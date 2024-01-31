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

package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// ForkExec forks a new process and execs the given command.
// The process will be terminated when the context is canceled.
func (c *Cluster) ForkExec(ctx context.Context, dir string, name string, args ...string) error {
	pidPath := path.Join(dir, "pids", path.OnlyName(name)+".pid")
	if file.Exists(pidPath) {
		pidData, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err := strconv.Atoi(string(pidData))
			if err == nil {
				if exec.IsRunning(pid) {
					return nil
				}
			}
		}
	}
	ctx = exec.WithDir(ctx, dir)
	ctx = exec.WithFork(ctx, true)
	logPath := path.Join(dir, "logs", path.OnlyName(name)+".log")
	logFile, err := c.OpenFile(logPath)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	ctx = exec.WithIOStreams(ctx, exec.IOStreams{
		Out:    logFile,
		ErrOut: logFile,
	})

	if c.IsDryRun() {
		dryrun.PrintMessage("%s", FormatExec(ctx, name, args...))
		dryrun.PrintMessage("echo $! >%s", pidPath)
		return nil
	}
	cmd, err := exec.Command(ctx, name, args...)
	if err != nil {
		return err
	}
	err = c.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)))
	if err != nil {
		return fmt.Errorf("write pid file %s: %w", pidPath, err)
	}

	return nil
}

// ForkExecKill kills the process if it is running.
func (c *Cluster) ForkExecKill(ctx context.Context, dir string, name string) error {
	pidPath := path.Join(dir, "pids", path.OnlyName(name)+".pid")
	if !file.Exists(pidPath) {
		// No pid file exists, which means the process has been terminated
		logger := log.FromContext(ctx)
		logger.Debug("Stat file not exists",
			"path", pidPath,
		)
		return nil
	}

	if c.IsDryRun() {
		dryrun.PrintMessage("kill $(cat %s)", pidPath)
	} else {
		raw, err := os.ReadFile(pidPath)
		if err != nil {
			return fmt.Errorf("read pid file %s: %w", pidPath, err)
		}
		pid, err := strconv.Atoi(string(raw))
		if err != nil {
			return fmt.Errorf("parse pid file %s: %w", pidPath, err)
		}
		err = exec.KillProcess(pid)
		if err != nil {
			return err
		}
	}
	err := c.Remove(pidPath)
	if err != nil {
		return err
	}

	return nil
}

// ForkExecIsRunning checks if the process is running.
func (c *Cluster) ForkExecIsRunning(ctx context.Context, dir string, name string) bool {
	pidPath := path.Join(dir, "pids", path.OnlyName(name)+".pid")
	if !file.Exists(pidPath) {
		logger := log.FromContext(ctx)
		logger.Debug("Stat file not exists",
			"path", pidPath,
		)
		return false
	}

	pidData, err := os.ReadFile(pidPath)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error("Read pid file", err)
		return false
	}
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false
	}
	return exec.IsRunning(pid)
}

// PullImages is a helper function to pull images
func (c *Cluster) PullImages(ctx context.Context, command string, images []string, quiet bool) error {
	if c.IsDryRun() {
		for _, image := range images {
			dryrun.PrintMessage("%s pull %s", command, image)
		}
		return nil
	}

	return exec.PullImages(ctx, command, images, quiet)
}

// EnsureImage ensures the image exists.
func (c *Cluster) EnsureImage(ctx context.Context, command string, image string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("%s pull %s", command, image)
		return nil
	}

	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := config.Options

	return exec.PullImage(ctx, command, image, conf.QuietPull)
}

// Exec executes the given command and returns the output.
func (c *Cluster) Exec(ctx context.Context, name string, args ...string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("%s", FormatExec(ctx, name, args...))
		return nil
	}

	return exec.Exec(ctx, name, args...)
}

// ParseVersionFromBinary parses the version from the given binary.
func (c *Cluster) ParseVersionFromBinary(ctx context.Context, path string) (version.Version, error) {
	if c.IsDryRun() {
		return version.Unknown, nil
	}

	return exec.ParseVersionFromBinary(ctx, path)
}

// ParseVersionFromImage parses the version from the image.
func (c *Cluster) ParseVersionFromImage(ctx context.Context, runtime string, image string, command string) (version.Version, error) {
	if c.IsDryRun() {
		return version.Unknown, nil
	}

	return exec.ParseVersionFromImage(ctx, runtime, image, command)
}

// WriteToPath writes the output of a command to a specified file
func (c *Cluster) WriteToPath(ctx context.Context, path string, commands []string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("%s >%s", strings.Join(commands, " "), path)
		return nil
	}

	buf := bytes.NewBuffer(nil)
	err := exec.Exec(exec.WithAllWriteTo(ctx, buf), commands[0], commands[1:]...)
	if err != nil {
		return err
	}

	return file.Write(path, buf.Bytes())
}

// FormatExec prints the command to be executed to the output stream.
func FormatExec(ctx context.Context, name string, args ...string) string {
	opt := exec.GetExecOptions(ctx)
	out := bytes.NewBuffer(nil)
	if opt.Dir != "" {
		_, _ = fmt.Fprintf(out, "cd %s && ", opt.Dir)
	}

	if len(opt.Env) != 0 {
		_, _ = fmt.Fprintf(out, "%s ", strings.Join(opt.Env, " "))
	}

	_, _ = fmt.Fprintf(out, "%s", path.OnlyName(name))

	for _, arg := range args {
		_, _ = fmt.Fprintf(out, " %s", arg)
	}

	outfile, ok := dryrun.IsCatToFileWriter(opt.Out)
	if ok {
		_, _ = fmt.Fprintf(out, " >%s", outfile)
	}

	if erroutfile, ok := dryrun.IsCatToFileWriter(opt.ErrOut); ok {
		if erroutfile == outfile {
			_, _ = fmt.Fprintf(out, " 2>&1")
		} else {
			_, _ = fmt.Fprintf(out, " 2>%s", outfile)
		}
	}

	if opt.Fork {
		_, _ = fmt.Fprintf(out, " &")
	}
	return out.String()
}
