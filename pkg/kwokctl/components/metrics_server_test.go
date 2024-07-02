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

	"github.com/blang/semver/v4"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
)

func TestBuildMetricsServerComponent(t *testing.T) {
	type args struct {
		conf BuildMetricsServerComponentConfig
	}
	tests := []struct {
		name          string
		args          args
		wantComponent internalversion.Component
		wantErr       bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComponent, err := BuildMetricsServerComponent(tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildMetricsServerComponent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotComponent, tt.wantComponent) {
				t.Errorf("BuildMetricsServerComponent() = %v, want %v", gotComponent, tt.wantComponent)
			}
		})
	}
}

// Handles empty or zero conf.Port correctly
func TestBuildMetricsServerComponent_HandlesEmptyOrZeroPort(t *testing.T) {
	conf := BuildMetricsServerComponentConfig{
		Runtime:        "container",
		Port:           0,
		BindAddress:    "127.0.0.1",
		KubeconfigPath: "/path/to/kubeconfig",
		CaCertPath:     "/path/to/ca.crt",
		AdminCertPath:  "/path/to/admin.crt",
		AdminKeyPath:   "/path/to/admin.key",
		Verbosity:      log.LevelInfo,
		Version:        semver.MustParse("1.0.0"),
		Binary:         "/usr/local/bin/metrics-server",
		Image:          "metrics-server:latest",
		Workdir:        "/var/lib/metrics-server",
	}

	component, err := BuildMetricsServerComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(component.Ports) != 0 {
		t.Errorf("expected no ports to be set, got %v", component.Ports)
	}
}

// Appends correct volumes for non-native runtime modes
func TestBuildMetricsServerComponent_AppendsCorrectVolumesForNonNativeRuntimeModes(t *testing.T) {
	conf := BuildMetricsServerComponentConfig{
		Runtime:        "container",
		Port:           8080,
		BindAddress:    "127.0.0.1",
		KubeconfigPath: "/path/to/kubeconfig",
		CaCertPath:     "/path/to/ca.crt",
		AdminCertPath:  "/path/to/admin.crt",
		AdminKeyPath:   "/path/to/admin.key",
		Verbosity:      log.LevelInfo,
		Version:        semver.MustParse("1.0.0"),
		Binary:         "/usr/local/bin/metrics-server",
		Image:          "metrics-server:latest",
		Workdir:        "/var/lib/metrics-server",
	}

	component, err := BuildMetricsServerComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedVolumes := []internalversion.Volume{
		{
			HostPath:  conf.KubeconfigPath,
			MountPath: "/root/.kube/config",
			ReadOnly:  true,
		},
		{
			HostPath:  conf.CaCertPath,
			MountPath: "/etc/kubernetes/pki/ca.crt",
			ReadOnly:  true,
		},
		{
			HostPath:  conf.AdminCertPath,
			MountPath: "/etc/kubernetes/pki/admin.crt",
			ReadOnly:  true,
		},
		{
			HostPath:  conf.AdminKeyPath,
			MountPath: "/etc/kubernetes/pki/admin.key",
			ReadOnly:  true,
		},
	}

	if !reflect.DeepEqual(component.Volumes, expectedVolumes) {
		t.Errorf("expected volumes %v, got %v", expectedVolumes, component.Volumes)
	}
}
