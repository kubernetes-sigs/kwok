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

package kind

import (
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/utils/exec"
)

var runtimeBinary = "docker"

func init() {
	runtime.DefaultRegistry.Register(consts.RuntimeTypeKind, NewCluster)
	if _, err := exec.LookPath(runtimeBinary); err != nil {
		if !exec.IsNotFound(err) {
			panic(err)
		}

		if _, err := exec.LookPath("podman"); err != nil {
			panic(err)
		}
		runtimeBinary = "podman"
	}
}
