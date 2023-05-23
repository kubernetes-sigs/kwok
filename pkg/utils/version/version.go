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
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"
)

// Version represents a semver compatible version
type Version = semver.Version

// Unknown is the unknown version.
var Unknown = Version{
	// For the unknown version we consider it as the largest version.
	Major: 255,
}

// NewVersion creates a new version.
func NewVersion(major, minor, patch uint64) Version {
	return semver.Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

var versionRegexp = regexp.MustCompile(`(kubernetes|version):? v?(\d+\.\d+\.\d+\S*)`)
var jaegerVersionRegexp = regexp.MustCompile(`"(gitversion)":"v(\d+\.\d+\.\d+)"`)

// ParseFromOutput parses the version from the output.
func ParseFromOutput(s string) (Version, error) {
	s = strings.ToLower(s)
	var matches []string
	matches = versionRegexp.FindStringSubmatch(s)
	if len(matches) == 0 {
		// try match jaeger version msg
		matches = jaegerVersionRegexp.FindStringSubmatch(s)
		if len(matches) == 0 {
			return semver.Version{}, fmt.Errorf("failed to parse version from output: %q", s)
		}
	}
	v := matches[2]
	if strings.HasPrefix(v, "0.0.0") {
		return Unknown, nil
	}
	return semver.Parse(v)
}

// ParseVersion parses the version.
func ParseVersion(s string) (Version, error) {
	return semver.ParseTolerant(s)
}
