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
)

func TestInClusterEnvs(t *testing.T) {
	conf := InClusterConfig{
		Host: "10.0.0.1",
		Port: 6443,
	}

	envs := InClusterEnvs(conf)

	if len(envs) != 2 {
		t.Errorf("expected 2 envs, got %d", len(envs))
	}

	var foundHost, foundPort bool
	for _, env := range envs {
		if env.Name == "KUBERNETES_SERVICE_HOST" && env.Value == "10.0.0.1" {
			foundHost = true
		}
		if env.Name == "KUBERNETES_SERVICE_PORT" && env.Value == "6443" {
			foundPort = true
		}
	}

	if !foundHost {
		t.Error("expected KUBERNETES_SERVICE_HOST env with value 10.0.0.1")
	}
	if !foundPort {
		t.Error("expected KUBERNETES_SERVICE_PORT env with value 6443")
	}
}

func TestInClusterVolumes(t *testing.T) {
	conf := InClusterConfig{
		TokenHostPath:  "/path/to/token",
		CACertHostPath: "/path/to/ca.crt",
	}

	volumes := InClusterVolumes(conf)

	if len(volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(volumes))
	}

	var foundToken, foundCACert bool
	for _, vol := range volumes {
		if vol.Name == "serviceaccount-token" &&
			vol.HostPath == "/path/to/token" &&
			vol.MountPath == InClusterTokenPath &&
			vol.ReadOnly {
			foundToken = true
		}
		if vol.Name == "serviceaccount-ca" &&
			vol.HostPath == "/path/to/ca.crt" &&
			vol.MountPath == InClusterCACertPath &&
			vol.ReadOnly {
			foundCACert = true
		}
	}

	if !foundToken {
		t.Error("expected serviceaccount-token volume with correct mount path")
	}
	if !foundCACert {
		t.Error("expected serviceaccount-ca volume with correct mount path")
	}
}

func TestInClusterConstants(t *testing.T) {
	expectedSecretPath := "/var/run/secrets/kubernetes.io/serviceaccount"
	if InClusterSecretPath != expectedSecretPath {
		t.Errorf("expected InClusterSecretPath to be %s, got %s", expectedSecretPath, InClusterSecretPath)
	}

	expectedTokenPath := expectedSecretPath + "/token"
	if InClusterTokenPath != expectedTokenPath {
		t.Errorf("expected InClusterTokenPath to be %s, got %s", expectedTokenPath, InClusterTokenPath)
	}

	expectedCACertPath := expectedSecretPath + "/ca.crt"
	if InClusterCACertPath != expectedCACertPath {
		t.Errorf("expected InClusterCACertPath to be %s, got %s", expectedCACertPath, InClusterCACertPath)
	}
}
