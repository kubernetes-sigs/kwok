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
	"strings"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// ForkExec forks a new process and execs the given command.
// The process will be terminated when the context is canceled.
func (c *Cluster) ForkExec(ctx context.Context, dir string, name string, arg ...string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("%s %s &", name, strings.Join(arg, " "))
		return nil
	}

	return exec.ForkExec(ctx, dir, name, arg...)
}

// ForkExecKill kills the process if it is running.
func (c *Cluster) ForkExecKill(ctx context.Context, dir string, name string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("pkill %s", name)
		return nil
	}

	return exec.ForkExecKill(ctx, dir, name)
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

// Exec executes the given command and returns the output.
func (c *Cluster) Exec(ctx context.Context, name string, args ...string) error {
	if c.IsDryRun() {
		dryrun.PrintMessage("%s", exec.FormatExec(ctx, name, args...))
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
