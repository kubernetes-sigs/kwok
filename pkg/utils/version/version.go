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

package version

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
)

// Version represents a semver compatible version
type Version = semver.Version

var versionRegexp = regexp.MustCompile(`(kubernetes|version):? v?(\d+\.\d+\.\d+)`)

// Unknown is the unknown version.
var Unknown = Version{}

// NewVersion creates a new version.
func NewVersion(major, minor, patch uint64) Version {
	return semver.Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

// ParseVersion parses the version.
func ParseVersion(s string) (Version, error) {
	return semver.ParseTolerant(s)
}

// ParseFromOutput parses the version from the output.
func ParseFromOutput(s string) (Version, error) {
	s = strings.ToLower(s)
	matches := versionRegexp.FindStringSubmatch(s)
	if len(matches) == 0 {
		return semver.Version{}, fmt.Errorf("failed to parse version from output: %q", s)
	}
	return semver.Parse(matches[2])
}

// ParseFromBinary parses the version from the binary.
func ParseFromBinary(ctx context.Context, path string) (Version, error) {
	out := bytes.NewBuffer(nil)
	err := exec.Exec(exec.WithAllWriteTo(ctx, out), path, "--version")
	if err != nil {
		return Version{}, err
	}
	logger := log.FromContext(ctx)
	content := out.String()
	ver, err := ParseFromOutput(content)
	if err != nil {
		logger.Warn("Failed to parse",
			"path", path,
			"output", content,
			"err", err,
		)
		return Unknown, nil
	}
	logger.Debug("Parsed version",
		"path", path,
		"version", ver,
	)
	return ver, nil
}

// ParseFromImage parses the version from the image.
func ParseFromImage(ctx context.Context, runtime string, image string, command string) (Version, error) {
	logger := log.FromContext(ctx)

	// Try to parse the version from the image tag.
	nameAndTag := strings.SplitN(image, ":", 2)
	if len(nameAndTag) == 2 {
		ver, err := semver.ParseTolerant(nameAndTag[1])
		if err == nil {
			logger.Debug("Parsed version",
				"image", image,
				"version", ver,
			)
			return ver, nil
		}
	}

	// Try to parse the version from the binary in the image.
	args := []string{"run", "--rm", image}
	if command != "" {
		args = append(args, command)
	}
	args = append(args, "--version")
	out := bytes.NewBuffer(nil)
	err := exec.Exec(exec.WithAllWriteTo(ctx, out), runtime, args...)
	if err != nil {
		return Version{}, err
	}
	content := out.String()
	ver, err := ParseFromOutput(content)
	if err != nil {
		logger.Warn("Failed to parse",
			"image", image,
			"output", content,
			"err", err,
		)
		return Unknown, nil
	}
	logger.Debug("Parsed version",
		"image", image,
		"version", ver,
	)
	return ver, nil
}
