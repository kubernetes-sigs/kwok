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
	"reflect"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// testNativePort is the port kwokctl allocates for a component in the native runtime.
const testNativePort = 32000

func TestBuildComponentReadinessProbe(t *testing.T) {
	builders := map[string]func(t *testing.T, runtime string) internalversion.Component{
		consts.ComponentKwokController: func(t *testing.T, runtime string) internalversion.Component {
			return BuildKwokControllerComponent(BuildKwokControllerComponentConfig{
				Runtime:     runtime,
				ProjectName: "kwok-test",
				BindAddress: "0.0.0.0",
				Port:        testNativePort,
			})
		},
		consts.ComponentEtcd: func(t *testing.T, runtime string) internalversion.Component {
			component, err := BuildEtcdComponent(BuildEtcdComponentConfig{
				Runtime:          runtime,
				ProjectName:      "kwok-test",
				Version:          version.NewVersion(3, 6, 10),
				BindAddress:      "0.0.0.0",
				Port:             testNativePort,
				PeerPort:         testNativePort + 1,
				QuotaBackendSize: "8Gi",
			})
			if err != nil {
				t.Fatal(err)
			}
			return component
		},
		consts.ComponentPrometheus: func(t *testing.T, runtime string) internalversion.Component {
			component, err := BuildPrometheusComponent(BuildPrometheusComponentConfig{
				Runtime:     runtime,
				BindAddress: "0.0.0.0",
				Port:        testNativePort,
			})
			if err != nil {
				t.Fatal(err)
			}
			return component
		},
		consts.ComponentKubeApiserverInsecureProxy: func(t *testing.T, runtime string) internalversion.Component {
			component, err := BuildKubectlProxyComponent(BuildKubectlProxyComponentConfig{
				Runtime:     runtime,
				ProjectName: "kwok-test",
				BindAddress: "0.0.0.0",
				Port:        testNativePort,
			})
			if err != nil {
				t.Fatal(err)
			}
			return component
		},
		consts.ComponentJaeger: func(t *testing.T, runtime string) internalversion.Component {
			component, err := BuildJaegerComponent(BuildJaegerComponentConfig{
				Runtime:      runtime,
				BindAddress:  "0.0.0.0",
				Port:         testNativePort,
				OtlpGrpcPort: testNativePort + 1,
			})
			if err != nil {
				t.Fatal(err)
			}
			return component
		},
	}

	tests := []struct {
		component string
		runtime   string
		want      *internalversion.ComponentProbe
	}{
		{consts.ComponentKwokController, consts.RuntimeTypeBinary, httpGetReadinessProbe("/healthz", testNativePort)},
		{consts.ComponentKwokController, consts.RuntimeTypeDocker, httpGetReadinessProbe("/healthz", 10247)},
		{consts.ComponentKwokController, consts.RuntimeTypeKind, httpGetReadinessProbe("/healthz", 10247)},

		// kind runs the etcd that kubeadm manages, so the etcd builder is only used by binary and compose.
		{consts.ComponentEtcd, consts.RuntimeTypeBinary, httpGetReadinessProbe("/health", testNativePort)},
		{consts.ComponentEtcd, consts.RuntimeTypeDocker, httpGetReadinessProbe("/health", 2379)},

		{consts.ComponentPrometheus, consts.RuntimeTypeBinary, httpGetReadinessProbe("/-/ready", testNativePort)},
		{consts.ComponentPrometheus, consts.RuntimeTypeDocker, httpGetReadinessProbe("/-/ready", 9090)},
		{consts.ComponentPrometheus, consts.RuntimeTypeKind, httpGetReadinessProbe("/-/ready", 9090)},

		{consts.ComponentKubeApiserverInsecureProxy, consts.RuntimeTypeBinary, httpGetReadinessProbe("/healthz", testNativePort)},
		{consts.ComponentKubeApiserverInsecureProxy, consts.RuntimeTypeDocker, httpGetReadinessProbe("/healthz", 8001)},
		{consts.ComponentKubeApiserverInsecureProxy, consts.RuntimeTypeKind, httpGetReadinessProbe("/healthz", 8001)},

		// The query endpoint only serves the UI, and the admin health endpoint is not configured.
		{consts.ComponentJaeger, consts.RuntimeTypeBinary, nil},
		{consts.ComponentJaeger, consts.RuntimeTypeDocker, nil},
		{consts.ComponentJaeger, consts.RuntimeTypeKind, nil},
	}
	for _, tt := range tests {
		t.Run(tt.component+"/"+tt.runtime, func(t *testing.T) {
			component := builders[tt.component](t, tt.runtime)
			if !reflect.DeepEqual(component.ReadinessProbe, tt.want) {
				t.Fatalf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, tt.want)
			}
			if tt.want == nil {
				return
			}

			// The probe must target a port the component actually listens on.
			if !hasPort(component, tt.want.HTTPGet.Port) {
				t.Errorf("probe port %d is not one of the component ports %#v", tt.want.HTTPGet.Port, component.Ports)
			}
		})
	}
}

func hasPort(component internalversion.Component, port uint32) bool {
	for _, p := range component.Ports {
		if p.Port == port {
			return true
		}
	}
	return false
}

func TestConvertToPodComponentReadinessProbe(t *testing.T) {
	component := BuildKwokControllerComponent(BuildKwokControllerComponentConfig{
		Runtime:     consts.RuntimeTypeKind,
		ProjectName: "kwok-test",
		BindAddress: "0.0.0.0",
		Port:        10247,
	})
	pod := ConvertToPod(component)

	want := &corev1.Probe{
		// No host, so the kubelet probes the pod IP, which is the node IP on the host network.
		HTTPGet: &corev1.HTTPGetAction{
			Path:   "/healthz",
			Port:   intstr.FromInt32(10247),
			Scheme: corev1.URISchemeHTTP,
		},
		TimeoutSeconds: readinessProbeTimeoutSeconds,
		PeriodSeconds:  1,
	}
	got := pod.Spec.Containers[0].ReadinessProbe
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadinessProbe = %#v, want %#v", got, want)
	}
	if !pod.Spec.HostNetwork {
		t.Fatal("pod must use the host network for the probe to reach the component port")
	}
}

func TestBuildKubeApiserverComponentReadinessProbe(t *testing.T) {
	https := func(path string, port uint32) *internalversion.ComponentProbe {
		probe := httpGetReadinessProbe(path, port)
		probe.HTTPGet.Scheme = internalversion.URISchemeHTTPS
		return probe
	}
	tests := []struct {
		name          string
		runtime       string
		version       version.Version
		secure        bool
		authorization bool
		want          *internalversion.ComponentProbe
		wantArg       string
	}{
		{
			name:          "readyz on the secure container port",
			runtime:       consts.RuntimeTypeDocker,
			version:       version.NewVersion(1, 37, 0),
			secure:        true,
			authorization: true,
			want:          https("/readyz", 6443),
			wantArg:       "--secure-port=6443",
		},
		{
			name:          "readyz on the secure native port",
			runtime:       consts.RuntimeTypeBinary,
			version:       version.NewVersion(1, 37, 0),
			secure:        true,
			authorization: true,
			want:          https("/readyz", testNativePort),
			wantArg:       "--secure-port=32000",
		},
		{
			name:          "first release with readyz",
			runtime:       consts.RuntimeTypeDocker,
			version:       version.NewVersion(1, 16, 0),
			secure:        true,
			authorization: true,
			want:          https("/readyz", 6443),
			wantArg:       "--secure-port=6443",
		},
		{
			name:          "healthz before readyz",
			runtime:       consts.RuntimeTypeDocker,
			version:       version.NewVersion(1, 15, 12),
			secure:        true,
			authorization: true,
			want:          https("/healthz", 6443),
			wantArg:       "--secure-port=6443",
		},
		{
			name:          "healthz on the secure native port before readyz",
			runtime:       consts.RuntimeTypeBinary,
			version:       version.NewVersion(1, 13, 12),
			secure:        true,
			authorization: true,
			want:          https("/healthz", testNativePort),
			wantArg:       "--secure-port=32000",
		},
		{
			name:    "insecure container port",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 12, 10),
			want:    httpGetReadinessProbe("/healthz", 8080),
			wantArg: "--insecure-port=8080",
		},
		{
			name:    "insecure native port",
			runtime: consts.RuntimeTypeBinary,
			version: version.NewVersion(1, 10, 13),
			want:    httpGetReadinessProbe("/healthz", testNativePort),
			wantArg: "--insecure-port=32000",
		},
		{
			name:    "insecure port with readyz",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 19, 16),
			want:    httpGetReadinessProbe("/readyz", 8080),
			wantArg: "--insecure-port=8080",
		},
		{
			// Anonymous requests are rejected with the AlwaysAllow authorizer, so the probe could never succeed.
			name:    "secure port without authorization",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 37, 0),
			secure:  true,
			want:    nil,
		},
		{
			name:    "secure native port without authorization",
			runtime: consts.RuntimeTypeBinary,
			version: version.NewVersion(1, 16, 15),
			secure:  true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, err := BuildKubeApiserverComponent(BuildKubeApiserverComponentConfig{
				Runtime:           tt.runtime,
				ProjectName:       "kwok-test",
				Version:           tt.version,
				BindAddress:       "0.0.0.0",
				Port:              testNativePort,
				EtcdAddress:       "127.0.0.1",
				SecurePort:        tt.secure,
				KubeAuthorization: tt.authorization,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(component.ReadinessProbe, tt.want) {
				t.Fatalf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, tt.want)
			}
			if tt.want == nil {
				return
			}

			// The probe must target the port kube-apiserver serves on, with the scheme it serves.
			if !hasPort(component, tt.want.HTTPGet.Port) {
				t.Errorf("probe port %d is not one of the component ports %#v", tt.want.HTTPGet.Port, component.Ports)
			}
			if !slices.Contains(component.Args, tt.wantArg) {
				t.Errorf("args %q do not contain %q", component.Args, tt.wantArg)
			}
		})
	}
}
