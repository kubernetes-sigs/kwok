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

package k8s

import (
	"os"
	"runtime"
)

// lists from https://github.com/kubernetes/kubernetes/blob/2d7dcf928c3e0e8dd4c29c421893a299e1a1b857/cmd/kubeadm/app/constants/constants.go#L491
var etcdVersions = map[int]string{
	8:  "3.0.17",
	9:  "3.1.12",
	10: "3.1.12",
	11: "3.2.18",
	12: "3.2.24",
	13: "3.2.24",
	14: "3.3.10",
	15: "3.3.10",
	16: "3.3.17-0",
	17: "3.4.3-0",
	18: "3.4.3-0",
	19: "3.4.13-0",
	20: "3.4.13-0",
	21: "3.4.13-0",
	22: "3.5.4-0",
	23: "3.5.4-0",
	24: "3.5.4-0",
	25: "3.5.4-0",
}

func GetEtcdVersion(version int) string {
	if version < 8 {
		version = 8
	}
	if version > 25 {
		version = 25
	}
	return etcdVersions[version]
}

func init() {
	os.Setenv("ETCD_UNSUPPORTED_ARCH", runtime.GOARCH)
	os.Setenv("ETCDCTL_API", "3")
}
