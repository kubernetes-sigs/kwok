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
	"fmt"
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

// Without paths
func TestBuildDashboardMetricsScraperWithoutPaths(t *testing.T) {
	conf := BuildDashboardMetricsScraperComponentConfig{
		Runtime:        "native",
		KubeconfigPath: "",
		Image:          "test-image",
		Workdir:        "/workdir",
	}
	component, err := BuildDashboardMetricsScraperComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	fmt.Printf("%+v\n", component)

	expectedArgs := internalversion.Component{
		Name:  "dashboard-metrics-scraper",
		Links: []string{"metrics-server"},
		Image: "test-image",
		User:  "root",
		Args: []string{
			"--db-file=/metrics.db",
			"--kubeconfig=/root/.kube/config",
		},
		WorkDir: "/workdir",
		Volumes: []internalversion.Volume{{ReadOnly: true,
			MountPath: "/root/.kube/config"},

			{ReadOnly: true, MountPath: "/etc/kubernetes/pki/ca.crt"},
			{ReadOnly: true,
				MountPath: "/etc/kubernetes/pki/admin.crt",
			},
			{ReadOnly: true, MountPath: "/etc/kubernetes/pki/admin.key"},
		},
	}

	if !reflect.DeepEqual(component, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component)
	}
}

// With paths
func TestBuildDashboardMetricsScraperWithPaths(t *testing.T) {
	conf := BuildDashboardMetricsScraperComponentConfig{
		Runtime:        "containerd",
		Binary:         "binary",
		Image:          "dashboard-image",
		Workdir:        "/workdir",
		CaCertPath:     "/path/to/ca.crt",
		AdminCertPath:  "/path/to/admin.crt",
		AdminKeyPath:   "/path/to/admin.key",
		KubeconfigPath: "/path/to/kubeconfig",
	}
	component, err := BuildDashboardMetricsScraperComponent(conf)
	fmt.Printf("%+v\n", component)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := internalversion.Component{
		Name:  "dashboard-metrics-scraper",
		Links: []string{"metrics-server"},
		Image: "dashboard-image",
		User:  "root",
		Args: []string{
			"--db-file=/metrics.db",
			"--kubeconfig=/root/.kube/config",
		},
		WorkDir: "/workdir",
		Volumes: []internalversion.Volume{
			{
				ReadOnly:  true,
				HostPath:  "/path/to/kubeconfig",
				MountPath: "/root/.kube/config",
			},

			{
				ReadOnly:  true,
				HostPath:  "/path/to/ca.crt",
				MountPath: "/etc/kubernetes/pki/ca.crt",
			},
			{
				ReadOnly:  true,
				HostPath:  "/path/to/admin.crt",
				MountPath: "/etc/kubernetes/pki/admin.crt",
			},
			{
				ReadOnly:  true,
				HostPath:  "/path/to/admin.key",
				MountPath: "/etc/kubernetes/pki/admin.key",
			},
		},
	}

	if !reflect.DeepEqual(component, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component)
	}
}

// Handles empty or nil config paths gracefully
func TestBuildDashboardMetricsScraperComponent_EmptyConfigPaths(t *testing.T) {
	conf := BuildDashboardMetricsScraperComponentConfig{
		Runtime:        "native",
		KubeconfigPath: "",
		Image:          "test-image",
		Workdir:        "/workdir",
	}

	component, err := BuildDashboardMetricsScraperComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--db-file=/metrics.db",
		"--kubeconfig=/root/.kube/config",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}
}

// Creates component with correct default arguments for native runtime
func TestBuildDashboardMetricsScraperComponent_NativeRuntime(t *testing.T) {
	conf := BuildDashboardMetricsScraperComponentConfig{
		Runtime:        "nnative",
		KubeconfigPath: "/path/to/kubeconfig",
		Image:          "test-image",
		Workdir:        "/workdir",
	}

	component, err := BuildDashboardMetricsScraperComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--db-file=/metrics.db",
		"--kubeconfig=/root/.kube/config",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}
}
