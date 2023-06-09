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

package path

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kwok/pkg/consts"
)

var (
	homeDir string
	workDir string
)

func init() {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		homeDir = os.TempDir()
		workDir = Join(homeDir, consts.ProjectName)
	} else {
		homeDir = dir
		workDir = Join(homeDir, "."+consts.ProjectName)
	}
}

// Home returns the home directory of the current user or a temporary directory.
func Home() string {
	return homeDir
}

// WorkDir returns the current working directory.
func WorkDir() string {
	return workDir
}

// Expand expands absolute directory in file paths.
func Expand(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	if path[0] == '~' {
		home := Home()
		if len(path) == 1 {
			path = home
		} else if path[1] == '/' || path[1] == '\\' {
			path = Join(home, path[2:])
		}
	}

	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return Clean(p), nil
}

// RelFromHome returns a path relative to the home directory.
// If the path is not relative to the home directory, the original path is returned.
func RelFromHome(target string) string {
	rel, err := filepath.Rel(Home(), target)
	if err != nil {
		return target
	}
	if strings.HasPrefix(rel, "..") {
		return target
	}
	return Join("~", rel)
}

// Join is a wrapper around filepath.Join.
func Join(elem ...string) string {
	return Clean(filepath.Join(elem...))
}

// Dir is a wrapper around filepath.Dir.
func Dir(path string) string {
	return Clean(filepath.Dir(path))
}
