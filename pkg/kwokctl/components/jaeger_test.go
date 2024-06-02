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
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
)

// Returns correct component with default configuration
func TestBuildJaegerComponentWithDefaultConfig(t *testing.T) {
	conf := BuildJaegerComponentConfig{
		Runtime:     "default",
		Port:        16686,
		BindAddress: "0.0.0.0",
		Verbosity:   log.LevelInfo,
		Version:     semver.MustParse("1.0.0"),
		Binary:      "/usr/bin/jaeger",
		Image:       "jaegertracing/all-in-one:1.0.0",
		Workdir:     "/var/lib/jaeger",
	}

	component, err := BuildJaegerComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--collector.otlp.enabled=true",
		"--query.http-server.host-port=0.0.0.0:16686",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}

	if component.Name != consts.ComponentJaeger {
		t.Errorf("expected component name %s, got %s", consts.ComponentJaeger, component.Name)
	}

	if component.Version != "1.0.0" {
		t.Errorf("expected version %s, got %s", "1.0.0", component.Version)
	}
}

// Correctly sets ports when runtime mode is not native
func TestBuildJaegerComponentNonNativeRuntimePorts(t *testing.T) {
	conf := BuildJaegerComponentConfig{
		Runtime:     "non-native",
		Port:        16686,
		BindAddress: "0.0.0.0",
		Verbosity:   log.LevelInfo,
		Version:     semver.MustParse("1.0.0"),
		Binary:      "/usr/bin/jaeger",
		Image:       "jaegertracing/all-in-one:1.0.0",
		Workdir:     "/var/lib/jaeger",
	}

	component, err := BuildJaegerComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPorts := []internalversion.Port{
		{
			HostPort: 16686,
			Port:     16686,
		},
	}

	if !reflect.DeepEqual(component.Ports, expectedPorts) {
		t.Errorf("expected ports %v, got %v", expectedPorts, component.Ports)
	}
}

// Correctly sets ports when runtime mode is native
func TestBuildJaegerComponentNativeRuntimePorts(t *testing.T) {
	conf := BuildJaegerComponentConfig{
		Runtime:      "native",
		Port:         16686,
		BindAddress:  "0.0.0.0",
		OtlpGrpcPort: 14250,
		Verbosity:    log.LevelInfo,
		Version:      semver.MustParse("1.0.0"),
		Binary:       "/usr/bin/jaeger",
		Image:        "jaegertracing/all-in-one:1.0.0",
		Workdir:      "/var/lib/jaeger",
	}

	component, err := BuildJaegerComponent(conf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedArgs := []string{
		"--collector.otlp.enabled=true",
		"--query.http-server.host-port=0.0.0.0:16686",
	}

	if !reflect.DeepEqual(component.Args, expectedArgs) {
		t.Errorf("expected args %v, got %v", expectedArgs, component.Args)
	}
}
