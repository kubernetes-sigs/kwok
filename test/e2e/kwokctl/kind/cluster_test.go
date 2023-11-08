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

package kind_test

import (
	"runtime"
	"testing"

	"sigs.k8s.io/kwok/test/e2e"
)

func TestMultiCluster(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping test on non-linux platform")
	}

	f0 := e2e.CaseMultiCluster(kwokctlPath, logsDir, 1, baseArgs...).
		Feature()
	testEnv.Test(t, f0)
}
