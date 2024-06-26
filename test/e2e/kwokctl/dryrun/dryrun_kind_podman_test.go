/*
Copyright 2024 The Kubernetes Authors.

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

package dryrun_test

import (
	"testing"

	"sigs.k8s.io/kwok/test/e2e"
)

func TestKindPodmanDryRun(t *testing.T) {
	f0 := e2e.CaseDryrun(clusterName, kwokctlPath, rootDir, "kind-podman", updateTestdata).Feature()
	testEnv.Test(t, f0)
}

func TestKindPodmanDryRunWithExtra(t *testing.T) {
	f0 := e2e.CaseDryrunWithExtra(clusterName, kwokctlPath, rootDir, "kind-podman", updateTestdata).Feature()
	testEnv.Test(t, f0)
}

func TestKindPodmanDryRunWithVerbosity(t *testing.T) {
	f0 := e2e.CaseDryrunWithVerbosity(clusterName, kwokctlPath, rootDir, "kind-podman", updateTestdata).Feature()
	testEnv.Test(t, f0)
}
