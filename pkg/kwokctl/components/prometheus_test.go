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
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
)

// Correct volumes and ports are set when runtime mode is not native
func TestBuildPrometheusComponent_NonNativeRuntime(t *testing.T) {
	conf := BuildPrometheusComponentConfig{
		Runtime:       "docker",
		ConfigPath:    "/path/to/config",
		AdminCertPath: "/path/to/admin.crt",
		AdminKeyPath:  "/path/to/admin.key",
		Port:          9090,
		BindAddress:   "0.0.0.0",
		Verbosity:     log.LevelInfo,

		//Version:       "",
		Binary:  "/usr/local/bin/prometheus",
		Image:   "prom/prometheus:v2.26.0",
		Workdir: "/prometheus",
	}

	component, err := BuildPrometheusComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(component.Volumes) != 3 {
		t.Errorf("expected 3 volumes, got %d", len(component.Volumes))
	}

	if len(component.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(component.Ports))
	}

	expectedArgs := []string{
		"--config.file=/etc/prometheus/prometheus.yaml",
		"--web.listen-address=0.0.0.0:9090",
	}
	for _, arg := range expectedArgs {
		if !contains(component.Args, arg) {
			t.Errorf("expected argument %s not found in component args", arg)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Metric component is correctly configured
func TestBuildPrometheusComponent_MetricConfiguration(t *testing.T) {
	conf := BuildPrometheusComponentConfig{
		Runtime:       "docker",
		ConfigPath:    "/path/to/config",
		AdminCertPath: "/path/to/admin.crt",
		AdminKeyPath:  "/path/to/admin.key",
		Port:          9090,
		BindAddress:   "0.0.0.0",
		Verbosity:     log.LevelInfo,
		Binary:        "/usr/local/bin/prometheus",
		Image:         "prom/prometheus:v2.26.0",
		Workdir:       "/prometheus",
	}

	component, err := BuildPrometheusComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedMetric := &internalversion.ComponentMetric{
		Scheme: "http",
		Host:   net.LocalAddress + ":" + format.String(conf.Port),
		Path:   "/metrics",
	}

	if !reflect.DeepEqual(component.Metric, expectedMetric) {
		t.Errorf("expected metric configuration does not match")
	}
}

// Correct arguments are set when runtime mode is native
func TestBuildPrometheusComponent_NativeRuntime(t *testing.T) {
	conf := BuildPrometheusComponentConfig{
		Runtime:     "native",
		ConfigPath:  "/path/to/config",
		Port:        9090,
		BindAddress: "0.0.0.0",
		Verbosity:   log.LevelInfo,
		Binary:      "/usr/local/bin/prometheus",
		Image:       "prom/prometheus:v2.26.0",
		Workdir:     "/prometheus",
	}

	component, err := BuildPrometheusComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--web.listen-address=0.0.0.0:9090",
	}
	for _, arg := range expectedArgs {
		if !contains(component.Args, arg) {
			t.Errorf("expected argument %s not found in component args", arg)
		}
	}
}
