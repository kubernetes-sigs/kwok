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

func httpsReadinessProbe(path string, port uint32) *internalversion.ComponentProbe {
	probe := httpGetReadinessProbe(path, port)
	probe.HTTPGet.Scheme = internalversion.URISchemeHTTPS
	return probe
}

func TestBuildKubeControllerManagerComponentReadinessProbe(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		version version.Version
		secure  bool
		want    *internalversion.ComponentProbe
		wantArg string
	}{
		{
			name:    "secure container port",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 37, 0),
			secure:  true,
			want:    httpsReadinessProbe("/healthz", 10257),
			wantArg: "--secure-port=10257",
		},
		{
			name:    "secure native port",
			runtime: consts.RuntimeTypeBinary,
			version: version.NewVersion(1, 37, 0),
			secure:  true,
			want:    httpsReadinessProbe("/healthz", testNativePort),
			wantArg: "--secure-port=32000",
		},
		{
			name:    "first release that allows the health paths",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 13, 0),
			secure:  true,
			want:    httpsReadinessProbe("/healthz", 10257),
			wantArg: "--secure-port=10257",
		},
		{
			// Without --authorization-always-allow-paths, an anonymous probe is not authorized.
			name:    "secure port before the health paths are allowed",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 12, 10),
			secure:  true,
			want:    nil,
		},
		{
			name:    "insecure container port",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 12, 10),
			want:    httpGetReadinessProbe("/healthz", 10252),
			wantArg: "--port=10252",
		},
		{
			name:    "insecure native port",
			runtime: consts.RuntimeTypeBinary,
			version: version.NewVersion(1, 10, 13),
			want:    httpGetReadinessProbe("/healthz", testNativePort),
			wantArg: "--port=32000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, err := BuildKubeControllerManagerComponent(BuildKubeControllerManagerComponentConfig{
				Runtime:           tt.runtime,
				ProjectName:       "kwok-test",
				Version:           tt.version,
				BindAddress:       "0.0.0.0",
				Port:              testNativePort,
				SecurePort:        tt.secure,
				KubeAuthorization: tt.secure,
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
			if !hasPort(component, tt.want.HTTPGet.Port) {
				t.Errorf("probe port %d is not one of the component ports %#v", tt.want.HTTPGet.Port, component.Ports)
			}
			if !slices.Contains(component.Args, tt.wantArg) {
				t.Errorf("args %q do not contain %q", component.Args, tt.wantArg)
			}
		})
	}
}

func TestBuildKubeSchedulerComponentReadinessProbe(t *testing.T) {
	before := previousRelease(kubeSchedulerReadyzVersion)
	tests := []struct {
		name       string
		runtime    string
		version    version.Version
		secure     bool
		configPath string
		want       *internalversion.ComponentProbe
		wantArg    string
	}{
		{
			name:    "readyz on the secure container port",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 37, 0),
			secure:  true,
			want:    httpsReadinessProbe("/readyz", 10259),
			wantArg: "--secure-port=10259",
		},
		{
			name:    "first release with readyz",
			runtime: consts.RuntimeTypeBinary,
			version: kubeSchedulerReadyzVersion,
			secure:  true,
			want:    httpsReadinessProbe("/readyz", testNativePort),
			wantArg: "--secure-port=32000",
		},
		{
			name:    "healthz before readyz",
			runtime: consts.RuntimeTypeDocker,
			version: before,
			secure:  true,
			want:    httpsReadinessProbe("/healthz", 10259),
			wantArg: "--secure-port=10259",
		},
		{
			name:    "oldest secure release",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 13, 12),
			secure:  true,
			want:    httpsReadinessProbe("/healthz", 10259),
			wantArg: "--secure-port=10259",
		},
		{
			// The scheduler-plugins image reports its own version, and serves /healthz.
			name:    "scheduler-plugins image",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(0, 34, 7),
			secure:  true,
			want:    httpsReadinessProbe("/healthz", 10259),
			wantArg: "--secure-port=10259",
		},
		{
			name:       "secure port with a configuration file",
			runtime:    consts.RuntimeTypeDocker,
			version:    version.NewVersion(1, 37, 0),
			secure:     true,
			configPath: "/scheduler.yaml",
			want:       httpsReadinessProbe("/readyz", 10259),
			wantArg:    "--secure-port=10259",
		},
		{
			name:    "insecure container port",
			runtime: consts.RuntimeTypeDocker,
			version: version.NewVersion(1, 12, 10),
			want:    httpGetReadinessProbe("/healthz", 10251),
			wantArg: "--port=10251",
		},
		{
			name:    "insecure native port",
			runtime: consts.RuntimeTypeBinary,
			version: version.NewVersion(1, 10, 13),
			want:    httpGetReadinessProbe("/healthz", testNativePort),
			wantArg: "--port=32000",
		},
		{
			// The insecure serving flags are ignored with a configuration file.
			name:       "insecure port with a configuration file",
			runtime:    consts.RuntimeTypeDocker,
			version:    version.NewVersion(1, 12, 10),
			configPath: "/scheduler.yaml",
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, err := BuildKubeSchedulerComponent(BuildKubeSchedulerComponentConfig{
				Runtime:     tt.runtime,
				ProjectName: "kwok-test",
				Version:     tt.version,
				BindAddress: "0.0.0.0",
				Port:        testNativePort,
				SecurePort:  tt.secure,
				ConfigPath:  tt.configPath,
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
			if !hasPort(component, tt.want.HTTPGet.Port) {
				t.Errorf("probe port %d is not one of the component ports %#v", tt.want.HTTPGet.Port, component.Ports)
			}
			if !slices.Contains(component.Args, tt.wantArg) {
				t.Errorf("args %q do not contain %q", component.Args, tt.wantArg)
			}
		})
	}
}

func TestBuildAddonComponentReadinessProbe(t *testing.T) {
	tests := []struct {
		name  string
		build func(v version.Version, healthProbePort uint32) (internalversion.Component, error)
		// verified is the earliest release whose /readyz was verified.
		verified version.Version
	}{
		{
			name: consts.ComponentKueue,
			build: func(v version.Version, port uint32) (internalversion.Component, error) {
				return BuildKueueComponent(BuildKueueComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: v, HealthProbePort: port})
			},
			verified: version.NewVersion(0, 18, 1),
		},
		{
			name: consts.ComponentLWS,
			build: func(v version.Version, port uint32) (internalversion.Component, error) {
				return BuildLWSComponent(BuildLWSComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: v, HealthProbePort: port})
			},
			verified: version.NewVersion(0, 9, 0),
		},
		{
			name: consts.ComponentNodeReadinessController,
			build: func(v version.Version, _ uint32) (internalversion.Component, error) {
				return BuildNodeReadinessControllerComponent(BuildNodeReadinessControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: v})
			},
			verified: version.NewVersion(0, 3, 0),
		},
		{
			name: consts.ComponentSchedulerPlugins,
			build: func(v version.Version, _ uint32) (internalversion.Component, error) {
				return BuildSchedulerPluginsControllerComponent(BuildSchedulerPluginsControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: v})
			},
			verified: version.NewVersion(0, 34, 7),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, c := range []struct {
				version version.Version
				want    *internalversion.ComponentProbe
			}{
				{tt.verified, httpGetReadinessProbe("/readyz", 8081)},
				{version.NewVersion(tt.verified.Major, tt.verified.Minor+1, 0), httpGetReadinessProbe("/readyz", 8081)},
				// An earlier release is not probed.
				{previousRelease(tt.verified), nil},
				// An unknown version is taken as the latest, as it is everywhere in kwokctl.
				{version.Unknown, httpGetReadinessProbe("/readyz", 8081)},
			} {
				component, err := tt.build(c.version, 8081)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(component.ReadinessProbe, c.want) {
					t.Errorf("version %s: ReadinessProbe = %#v, want %#v", c.version, component.ReadinessProbe, c.want)
				}
			}
		})
	}

	// The port comes from the configuration, and a configuration that turns the probes off is not probed.
	component, err := BuildKueueComponent(BuildKueueComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 18, 1), HealthProbePort: 9081})
	if err != nil {
		t.Fatal(err)
	}
	if want := httpGetReadinessProbe("/readyz", 9081); !reflect.DeepEqual(component.ReadinessProbe, want) {
		t.Errorf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, want)
	}
	component, err = BuildLWSComponent(BuildLWSComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 9, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if component.ReadinessProbe != nil {
		t.Errorf("ReadinessProbe = %#v, want none without a health probe port", component.ReadinessProbe)
	}
}

// previousRelease returns the release before v.
func previousRelease(v version.Version) version.Version {
	if v.Patch > 0 {
		return version.NewVersion(v.Major, v.Minor, v.Patch-1)
	}
	return version.NewVersion(v.Major, v.Minor-1, 0)
}

func TestHealthProbePortFromConfig(t *testing.T) {
	tests := []struct {
		config string
		want   uint32
	}{
		{config: "apiVersion: config.kueue.x-k8s.io/v1beta2\nkind: Configuration\nhealth:\n  healthProbeBindAddress: :8081\n", want: 8081},
		{config: "health:\n  healthProbeBindAddress: 0.0.0.0:9081\n", want: 9081},
		// Unset, the configuration API defaults it to :8081.
		{config: "apiVersion: config.lws.x-k8s.io/v1alpha1\nkind: Configuration\nleaderElection:\n  leaderElect: false\n", want: 8081},
		{config: "health: {}\n", want: 8081},
		// "0" turns the endpoint off.
		{config: "health:\n  healthProbeBindAddress: \"0\"\n", want: 0},
		{config: "health:\n  healthProbeBindAddress: \"\"\n", want: 0},
		{config: "health:\n  healthProbeBindAddress: :http\n", want: 0},
		{config: "health: [", want: 0},
	}
	for _, tt := range tests {
		if got := HealthProbePortFromConfig(tt.config); got != tt.want {
			t.Errorf("HealthProbePortFromConfig(%q) = %d, want %d", tt.config, got, tt.want)
		}
	}
}
