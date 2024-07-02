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
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
)

func TestBuildDashboardComponent(t *testing.T) {
	tests := []struct {
		name   string
		config BuildDashboardComponentConfig
		want   internalversion.Component
	}{
		{
			name: "default config with metrics enabled",
			config: BuildDashboardComponentConfig{
				Runtime:        "container",
				ProjectName:    "kwok",
				Image:          "dashboard-image",
				Workdir:        "/workdir",
				BindAddress:    "0.0.0.0",
				Port:           8080,
				Banner:         "Welcome",
				EnableMetrics:  true,
				CaCertPath:     "/path/to/ca.crt",
				AdminCertPath:  "/path/to/admin.crt",
				AdminKeyPath:   "/path/to/admin.key",
				KubeconfigPath: "/path/to/kubeconfig",
			},
			want: internalversion.Component{
				Name:  consts.ComponentDashboard,
				Image: "dashboard-image",
				Links: []string{
					consts.ComponentKubeApiserver,
				},
				WorkDir: "/workdir",
				Ports: []internalversion.Port{
					{
						Name:     "http",
						Port:     8080,
						HostPort: 8080,
						Protocol: internalversion.ProtocolTCP,
					},
				},
				Volumes: []internalversion.Volume{
					{
						HostPath:  "/path/to/kubeconfig",
						MountPath: "/root/.kube/config",
						ReadOnly:  true,
					},
					{
						HostPath:  "/path/to/ca.crt",
						MountPath: "/etc/kubernetes/pki/ca.crt",
						ReadOnly:  true,
					},
					{
						HostPath:  "/path/to/admin.crt",
						MountPath: "/etc/kubernetes/pki/admin.crt",
						ReadOnly:  true,
					},
					{
						HostPath:  "/path/to/admin.key",
						MountPath: "/etc/kubernetes/pki/admin.key",
						ReadOnly:  true,
					},
				},
				Args: []string{
					"--insecure-bind-address=0.0.0.0",
					"--bind-address=127.0.0.1",
					"--port=0",
					"--enable-insecure-login",
					"--enable-skip-login",
					"--disable-settings-authorizer",
					"--sidecar-host=127.0.0.1:8000",
					"--system-banner=Welcome",
					"--kubeconfig=/root/.kube/config",
					"--insecure-port=8080",
				},
				User: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildDashboardComponent(tt.config)
			if err != nil {
				t.Fatalf("BuildDashboardComponent() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildDashboardComponent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Generates correct dashboardArgs with default configuration
func TestBuildDashboardComponent_DefaultConfiguration(t *testing.T) {
	conf := BuildDashboardComponentConfig{
		BindAddress:    "0.0.0.0",
		EnableMetrics:  false,
		Runtime:        "native",
		Banner:         "",
		KubeconfigPath: "/path/to/kubeconfig",
		CaCertPath:     "/path/to/ca.crt",
		AdminCertPath:  "/path/to/admin.crt",
		AdminKeyPath:   "/path/to/admin.key",
		Port:           8080,
		Image:          "dashboard-image",
		Workdir:        "/workdir",
	}

	component, err := BuildDashboardComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--insecure-bind-address=0.0.0.0",
		"--bind-address=127.0.0.1",
		"--port=0",
		"--enable-insecure-login",
		"--enable-skip-login",
		"--disable-settings-authorizer",
		"--metrics-provider=none",
		"--kubeconfig=/root/.kube/config",
		"--insecure-port=8080",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}
}

// Handles empty BindAddress
func TestBuildDashboardComponent_EmptyBindAddress(t *testing.T) {
	conf := BuildDashboardComponentConfig{
		BindAddress:    "",
		EnableMetrics:  false,
		Runtime:        "native",
		Banner:         "",
		KubeconfigPath: "/path/to/kubeconfig",
		CaCertPath:     "/path/to/ca.crt",
		AdminCertPath:  "/path/to/admin.crt",
		AdminKeyPath:   "/path/to/admin.key",
		Port:           8080,
		Image:          "dashboard-image",
		Workdir:        "/workdir",
	}

	component, err := BuildDashboardComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--insecure-bind-address=",
		"--bind-address=127.0.0.1",
		"--port=0",
		"--enable-insecure-login",
		"--enable-skip-login",
		"--disable-settings-authorizer",
		"--metrics-provider=none",
		"--kubeconfig=/root/.kube/config",
		"--insecure-port=8080",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}
}
