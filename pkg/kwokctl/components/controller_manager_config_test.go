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

package components

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	utilyaml "sigs.k8s.io/kwok/pkg/utils/yaml"
)

func TestBuildKueueConfig(t *testing.T) {
	rawManifest := strings.TrimSpace(`
apiVersion: v1
kind: Namespace
metadata:
  name: kueue-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: kueue-manager-config
  namespace: kueue-system
data:
  controller_manager_config.yaml: |
    apiVersion: config.kueue.x-k8s.io/v1beta2
    kind: Configuration
    leaderElection:
      leaderElect: true
      resourceName: upstream
    webhook:
      port: 9443
`)

	got, err := BuildKueueConfig(rawManifest)
	if err != nil {
		t.Fatalf("BuildKueueConfig() error = %v", err)
	}

	config := map[string]any{}
	if err := utilyaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if diff := cmp.Diff("config.kueue.x-k8s.io/v1beta2", config["apiVersion"]); diff != "" {
		t.Fatalf("apiVersion diff (-want +got):\n%s", diff)
	}

	leaderElect, found, err := unstructured.NestedBool(config, "leaderElection", "leaderElect")
	if err != nil || !found {
		t.Fatalf("leaderElection.leaderElect not found: found=%v err=%v", found, err)
	}
	if leaderElect {
		t.Fatalf("leaderElection.leaderElect = true, want false")
	}

	internalCertManagement, found, err := unstructured.NestedBool(config, "internalCertManagement", "enable")
	if err != nil || !found {
		t.Fatalf("internalCertManagement.enable not found: found=%v err=%v", found, err)
	}
	if internalCertManagement {
		t.Fatalf("internalCertManagement.enable = true, want false")
	}
}

func TestBuildJobSetConfig(t *testing.T) {
	rawManifest := strings.TrimSpace(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: jobset-manager-config
  namespace: jobset-system
data:
  controller_manager_config.yaml: |
    apiVersion: config.jobset.x-k8s.io/v1alpha1
    kind: Configuration
    leaderElection:
      leaderElect: true
    internalCertManagement:
      enable: true
`)

	got, err := BuildJobSetConfig(rawManifest)
	if err != nil {
		t.Fatalf("BuildJobSetConfig() error = %v", err)
	}

	config := map[string]any{}
	if err := utilyaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	leaderElect, found, err := unstructured.NestedBool(config, "leaderElection", "leaderElect")
	if err != nil || !found {
		t.Fatalf("leaderElection.leaderElect not found: found=%v err=%v", found, err)
	}
	if leaderElect {
		t.Fatalf("leaderElection.leaderElect = true, want false")
	}

	internalCertManagement, found, err := unstructured.NestedBool(config, "internalCertManagement", "enable")
	if err != nil || !found {
		t.Fatalf("internalCertManagement.enable not found: found=%v err=%v", found, err)
	}
	if internalCertManagement {
		t.Fatalf("internalCertManagement.enable = true, want false")
	}
}

func TestBuildLWSConfig(t *testing.T) {
	rawManifest := strings.TrimSpace(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: lws-manager-config
  namespace: lws-system
data:
  controller_manager_config.yaml: |
    apiVersion: config.lws.x-k8s.io/v1alpha1
    kind: Configuration
    leaderElection:
      leaderElect: true
    internalCertManagement:
      enable: true
`)

	got, err := BuildLWSConfig(rawManifest)
	if err != nil {
		t.Fatalf("BuildLWSConfig() error = %v", err)
	}

	config := map[string]any{}
	if err := utilyaml.Unmarshal([]byte(got), &config); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	leaderElect, found, err := unstructured.NestedBool(config, "leaderElection", "leaderElect")
	if err != nil || !found {
		t.Fatalf("leaderElection.leaderElect not found: found=%v err=%v", found, err)
	}
	if leaderElect {
		t.Fatalf("leaderElection.leaderElect = true, want false")
	}

	internalCertManagement, found, err := unstructured.NestedBool(config, "internalCertManagement", "enable")
	if err != nil || !found {
		t.Fatalf("internalCertManagement.enable not found: found=%v err=%v", found, err)
	}
	if internalCertManagement {
		t.Fatalf("internalCertManagement.enable = true, want false")
	}
}
