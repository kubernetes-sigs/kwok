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

package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildKwokControllerComponent_ManageNodesWithAnnotationSelectorEmpty(t *testing.T) {
	conf := BuildKwokControllerComponentConfig{
		ManageNodesWithAnnotationSelector: "",
		Runtime:                           "native",
		KubeconfigPath:                    "/path/to/kubeconfig",
		CaCertPath:                        "/path/to/ca.crt",
		AdminCertPath:                     "/path/to/admin.crt",
		AdminKeyPath:                      "/path/to/admin.key",
		ConfigPath:                        "/path/to/config",
		NodeIP:                            "127.0.0.1",
		NodeName:                          "test-node",
		BindAddress:                       "0.0.0.0",
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        []string{},
		Binary:                            "kwok",
		Image:                             "kwok-image",
		Workdir:                           "/workdir",
	}

	component := BuildKwokControllerComponent(conf)

	assert.Contains(t, component.Args, "--manage-all-nodes=true")
	assert.NotContains(t, component.Args, "--manage-nodes-with-annotation-selector=")
}

func TestBuildKwokControllerComponent_EmptyPort(t *testing.T) {
	conf := BuildKwokControllerComponentConfig{
		ManageNodesWithAnnotationSelector: "key=value",
		Runtime:                           "native",
		KubeconfigPath:                    "/path/to/kubeconfig",
		CaCertPath:                        "/path/to/ca.crt",
		AdminCertPath:                     "/path/to/admin.crt",
		AdminKeyPath:                      "/path/to/admin.key",
		ConfigPath:                        "/path/to/config",
		NodeIP:                            "127.0.0.1",
		NodeName:                          "test-node",
		BindAddress:                       "0.0.0.0",
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        []string{},
		Binary:                            "kwok",
		Image:                             "kwok-image",
		Workdir:                           "/workdir",
	}

	component := BuildKwokControllerComponent(conf)

	assert.NotContains(t, component.Args, "--node-port=")
	assert.NotContains(t, component.Args, "--server-address=0.0.0.0:")
}

// Sets metricsHost correctly for native runtime mode
func TestBuildKwokControllerComponent_MetricsHostForNativeRuntime(t *testing.T) {
	conf := BuildKwokControllerComponentConfig{
		ManageNodesWithAnnotationSelector: "",
		Runtime:                           "native",
		KubeconfigPath:                    "/path/to/kubeconfig",
		CaCertPath:                        "/path/to/ca.crt",
		AdminCertPath:                     "/path/to/admin.crt",
		AdminKeyPath:                      "/path/to/admin.key",
		ConfigPath:                        "/path/to/config",
		NodeIP:                            "127.0.0.1",
		NodeName:                          "test-node",
		BindAddress:                       "0.0.0.0",
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        []string{},
		Binary:                            "kwok",
		Image:                             "kwok-image",
		Workdir:                           "/workdir",
		Port:                              8080,
	}

	component := BuildKwokControllerComponent(conf)

	assert.Contains(t, component.Args, "--node-port=10247")
	assert.Contains(t, component.Args, "--node-lease-duration-seconds=40")
	assert.Contains(t, component.Args, "--node-ip=127.0.0.1")
}

// Appends enable-crds argument when EnableCRDs is not empty
func TestBuildKwokControllerComponent_EnableCRDsNotEmpty(t *testing.T) {
	conf := BuildKwokControllerComponentConfig{
		ManageNodesWithAnnotationSelector: "",
		Runtime:                           "native",
		KubeconfigPath:                    "/path/to/kubeconfig",
		CaCertPath:                        "/path/to/ca.crt",
		AdminCertPath:                     "/path/to/admin.crt",
		AdminKeyPath:                      "/path/to/admin.key",
		ConfigPath:                        "/path/to/config",
		NodeIP:                            "127.0.0.1",
		NodeName:                          "test-node",
		BindAddress:                       "0.0.0.0",
		NodeLeaseDurationSeconds:          40,
		EnableCRDs:                        []string{"crd1", "crd2"},
		Binary:                            "kwok",
		Image:                             "kwok-image",
		Workdir:                           "/workdir",
	}

	component := BuildKwokControllerComponent(conf)

	assert.Contains(t, component.Args, "--enable-crds=crd1,crd2")
}
