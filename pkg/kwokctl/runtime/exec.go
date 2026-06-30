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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	utilsimage "sigs.k8s.io/kwok/pkg/utils/image"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// ForkExec forks a new process and execs the given command.
// The process will be terminated when the context is canceled.
func (c *Cluster) ForkExec(ctx context.Context, dir string, name string, args ...string) error {
	pidPath := utilspath.Join(dir, "pids", utilspath.OnlyName(name)+".pid")
	if file.Exists(pidPath) {
		pidData, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err := strconv.Atoi(string(pidData))
			if err == nil {
				if utilsexec.IsRunning(pid) {
					return nil
				}
			}
		}
	}
	ctx = utilsexec.WithDir(ctx, dir)
	ctx = utilsexec.WithFork(ctx, true)
	logPath := utilspath.Join(dir, "logs", utilspath.OnlyName(name)+".log")
	logFile, err := c.OpenFile(logPath)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}

	ctx = utilsexec.WithIOStreams(ctx, utilsexec.IOStreams{
		Out:    logFile,
		ErrOut: logFile,
	})

	if c.IsDryRun() {
		dryrun.PrintExec(ctx, name, args...)
		dryrun.PrintMessagef("echo $! >%s", pidPath)
		return nil
	}
	cmd, err := utilsexec.Command(ctx, name, args...)
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
	pidPath := utilspath.Join(dir, "pids", utilspath.OnlyName(name)+".pid")
	if !file.Exists(pidPath) {
		// No pid file exists, which means the process has been terminated
		logger := log.FromContext(ctx)
		logger.Debug("Stat file not exists",
			"path", pidPath,
		)
		return nil
	}

	if c.IsDryRun() {
		dryrun.PrintMessagef("kill $(cat %s)", pidPath)
	} else {
		raw, err := os.ReadFile(pidPath)
		if err != nil {
			return fmt.Errorf("read pid file %s: %w", pidPath, err)
		}
		pid, err := strconv.Atoi(string(raw))
		if err != nil {
			return fmt.Errorf("parse pid file %s: %w", pidPath, err)
		}
		err = utilsexec.KillProcess(pid)
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
	pidPath := utilspath.Join(dir, "pids", utilspath.OnlyName(name)+".pid")
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
		logger.Error("Read pid file",
			"err", err,
		)
		return false
	}
	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return false
	}
	return utilsexec.IsRunning(pid)
}

// EnsureImage ensures the image exists.
func (c *Cluster) EnsureImage(ctx context.Context, command string, image string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("%s pull %s", command, image)
		return nil
	}

	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := config.Options

	logger := log.FromContext(ctx)

	err = utilsexec.Exec(ctx,
		command, "inspect",
		image,
	)
	if err == nil {
		logger.Debug("Image already exists",
			"image", image,
		)
		return nil
	}

	err = c.ensureImage(ctx, command, image, conf.QuietPull, conf.CacheDir)
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		logger.Debug("Failed to pull",
			"image", image,
			"err", err,
		)
		err0 := c.ensureImageWithRuntime(ctx, command, image, conf.QuietPull)
		if err0 != nil {
			return errors.Join(err, err0)
		}
	}
	return nil
}

func (c *Cluster) ensureImage(ctx context.Context, command string, image string, quiet bool, cacheDir string) error {
	dest := utilspath.Join(cacheDir, "tarball", image+".tar")
	err := c.MkdirAll(filepath.Dir(dest))
	if err != nil {
		return err
	}
	cache := utilspath.Join(cacheDir, "blobs")
	err = utilsimage.Pull(ctx, cache, image, dest, quiet)
	if err != nil {
		return err
	}

	err = utilsexec.Exec(ctx, command, "load",
		"-i", dest,
	)
	if err != nil {
		return err
	}

	err = file.Remove(dest)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Error("Remove file",
			"err", err,
		)
	}
	return nil
}

func (c *Cluster) ensureImageWithRuntime(ctx context.Context, command string, image string, quiet bool) error {
	var out io.Writer = os.Stderr
	if quiet {
		out = nil
	}
	return utilsexec.Exec(utilsexec.WithAllWriteTo(ctx, out), command, "pull",
		image,
	)
}

// Exec executes the given command and returns the output.
func (c *Cluster) Exec(ctx context.Context, name string, args ...string) error {
	if c.IsDryRun() {
		dryrun.PrintExec(ctx, name, args...)
		return nil
	}

	return utilsexec.Exec(ctx, name, args...)
}

// ParseVersionFromBinary parses the version from the given binary.
func (c *Cluster) ParseVersionFromBinary(ctx context.Context, path string) (version.Version, error) {
	if c.IsDryRun() {
		return version.Unknown, nil
	}

	return utilsexec.ParseVersionFromBinary(ctx, path)
}

// ParseVersionFromImage parses the version from the image.
func (c *Cluster) ParseVersionFromImage(ctx context.Context, runtime string, image string, command string) (version.Version, error) {
	if c.IsDryRun() {
		return version.Unknown, nil
	}

	return utilsexec.ParseVersionFromImage(ctx, runtime, image, command)
}

// WriteToPath writes the output of a command to a specified file
func (c *Cluster) WriteToPath(ctx context.Context, path string, commands []string) error {
	out, err := c.OpenFile(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	err = c.Exec(utilsexec.WithAllWriteTo(ctx, out), commands[0], commands[1:]...)
	if err != nil {
		return err
	}
	return nil
}
