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

func TestBuildDashboardMetricsScraperComponent(t *testing.T) {
	type args struct {
		conf BuildDashboardMetricsScraperComponentConfig
	}
	tests := []struct {
		name          string
		args          args
		want          internalversion.Component
		wantComponent string

		wantErr bool
	}{{
		name: "as",
		args: args{
			conf: BuildDashboardMetricsScraperComponentConfig{
				Runtime:        "native",
				KubeconfigPath: "",
				Image:          "test-image",
				Workdir:        "/workdir",
			},
		},
		wantComponent: `{dashboard-metrics-scraper [metrics-server]  test-image [] root [--db-file=/metrics.db --kubeconfig=/root/.kube/config] /workdir [] [] [{ true  /root/.kube/config } { true  /etc/kubernetes/pki/ca.crt } { true  /etc/kubernetes/pki/admin.crt } { true  /etc/kubernetes/pki/admin.key }] <nil> <nil> }`,
	},
		{
			name: "ss",
			args: args{
				conf: BuildDashboardMetricsScraperComponentConfig{
					Runtime:        "containerd",
					Binary:         "binary",
					Image:          "dashboard-image",
					Workdir:        "/workdir",
					CaCertPath:     "/path/to/ca.crt",
					AdminCertPath:  "/path/to/admin.crt",
					AdminKeyPath:   "/path/to/admin.key",
					KubeconfigPath: "/path/to/kubeconfig",
				},
			},
			wantComponent: "{dashboard-metrics-scraper [metrics-server]  dashboard-image [] root [--db-file=/metrics.db --kubeconfig=/root/.kube/config] /workdir [] [] [{ true /path/to/kubeconfig /root/.kube/config } { true /path/to/ca.crt /etc/kubernetes/pki/ca.crt } { true /path/to/admin.crt /etc/kubernetes/pki/admin.crt } { true /path/to/admin.key /etc/kubernetes/pki/admin.key }] <nil> <nil> }"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComponent, err := BuildDashboardMetricsScraperComponent(tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildDashboardMetricsScraperComponent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			a := fmt.Sprintf("%v", gotComponent)
			if !reflect.DeepEqual(a, tt.wantComponent) {
				t.Errorf("BuildDashboardMetricsScraperComponent() = %v, want %v", gotComponent, a)
			}
		})
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
