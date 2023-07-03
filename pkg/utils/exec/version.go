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

package exec

import (
	"bytes"
	"context"
	"strings"

	"github.com/blang/semver/v4"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// ParseVersionFromBinary parses the version from the binary.
func ParseVersionFromBinary(ctx context.Context, path string) (version.Version, error) {
	out := bytes.NewBuffer(nil)
	err := Exec(WithAllWriteTo(ctx, out), path, "--version")
	if err != nil {
		return version.Version{}, err
	}
	logger := log.FromContext(ctx)
	content := out.String()
	ver, err := version.ParseFromOutput(content)
	if err != nil {
		logger.Warn("Failed to parse",
			"path", path,
			"output", content,
			"err", err,
		)
		return version.Unknown, nil
	}
	logger.Debug("Parsed version",
		"path", path,
		"version", ver,
	)
	return ver, nil
}

// ParseVersionFromImage parses the version from the image.
func ParseVersionFromImage(ctx context.Context, runtime string, image string, command string) (version.Version, error) {
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
	err := Exec(WithAllWriteTo(ctx, out), runtime, args...)
	if err != nil {
		return version.Version{}, err
	}
	content := out.String()
	ver, err := version.ParseFromOutput(content)
	if err != nil {
		logger.Warn("Failed to parse",
			"image", image,
			"output", content,
			"err", err,
		)
		return version.Unknown, nil
	}
	logger.Debug("Parsed version",
		"image", image,
		"version", ver,
	)
	return ver, nil
}
