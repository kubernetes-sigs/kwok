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

package default_test

import (
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"sigs.k8s.io/kwok/test/e2e"
)

func TestNode(t *testing.T) {
	f0 := e2e.CaseNode(envconf.RandomName("node", 16)).
		Feature()
	testEnv.Test(t, f0)
}

func TestPod(t *testing.T) {
	f0 := e2e.CasePod(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}

func TestStage(t *testing.T) {
	f0 := e2e.CaseStage().
		Feature()
	testEnv.Test(t, f0)
}

func TestJsonpatch(t *testing.T) {
	f0 := e2e.CaseJsonpatch(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}

func TestAttachMapping(t *testing.T) {
	f0 := e2e.CaseAttachMapping(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}

func TestExecMapping(t *testing.T) {
	f0 := e2e.CaseExecMapping(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}

func TestLogsMapping(t *testing.T) {
	f0 := e2e.CaseLogsMapping(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}

func TestPortForwardMapping(t *testing.T) {
	f0 := e2e.CasePortForwardMapping(envconf.RandomName("node", 16), namespace).
		Feature()
	testEnv.Test(t, f0)
}
