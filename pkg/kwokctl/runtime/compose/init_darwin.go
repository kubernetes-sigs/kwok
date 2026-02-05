/*
Copyright 2026 The Kubernetes Authors.

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

package compose

import (
	"bytes"

	"golang.org/x/sys/unix"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

func init() {
	if !isSupportAppleContainer() {
		return
	}
	runtime.DefaultRegistry.Register(consts.RuntimeTypeAppleContainer, NewAppleContainerCluster)
}

func isSupportAppleContainer() bool {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return false
	}

	if fromCString(uname.Sysname[:]) != "Darwin" {
		return false
	}

	release, err := version.ParseVersion(fromCString(uname.Release[:]))
	if err != nil {
		return false
	}
	if release.Major < 25 {
		return false
	}
	return true
}

func fromCString(cstr []byte) string {
	n := bytes.IndexByte(cstr, 0)
	if n == -1 {
		return string(cstr)
	}
	return string(cstr[:n])
}
