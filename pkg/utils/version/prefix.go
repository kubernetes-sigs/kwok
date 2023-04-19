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
	"strings"
)

// TrimPrefixV returns the version without the prefix 'v'.
func TrimPrefixV(version string) string {
	if len(version) <= 1 {
		return version
	}

	// Not a semantic version or unprefixed 'v'
	if version[0] != 'v' ||
		!strings.Contains(version, ".") ||
		version[1] < '0' ||
		version[1] > '9' {
		return version
	}
	return version[1:]
}

// AddPrefixV returns the version with the prefix 'v'.
func AddPrefixV(version string) string {
	if version == "" {
		return version
	}

	// Not a semantic version or prefixed 'v'
	if !strings.Contains(version, ".") ||
		version[0] < '0' ||
		version[0] > '9' {
		return version
	}
	return "v" + version
}
