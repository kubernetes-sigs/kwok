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

// Package consts defines the common variables.
package consts

import (
	"fmt"
	"runtime"
)

// Version information set by link flags during build. We fall back to these sane
// default values when we build outside the Makefile context (e.g. go run, go build, or go test).
var (
	version   = "99.99.99"             // value from VERSION file
	buildDate = "1970-01-01T00:00:00Z" // output from `date -u +'%Y-%m-%dT%H:%M:%SZ'`
)

// VersionInfo contains Kwok version information
type VersionInfo struct {
	Version   string
	BuildDate string
	GoVersion string
	Compiler  string
	Platform  string
}

func (v VersionInfo) String() string {
	return v.Version
}

// GetVersion returns the version information
func GetVersion() VersionInfo {
	return VersionInfo{
		Version:   version,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
