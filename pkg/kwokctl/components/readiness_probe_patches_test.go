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
	"errors"
	"reflect"
	"slices"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

func TestReconcileReadinessProbe(t *testing.T) {
	apiserver := func(t *testing.T, runtime string, secure, authorization bool) internalversion.Component {
		component, err := BuildKubeApiserverComponent(BuildKubeApiserverComponentConfig{
			Runtime:           runtime,
			ProjectName:       "kwok-test",
			Version:           version.NewVersion(1, 37, 0),
			BindAddress:       "0.0.0.0",
			Port:              testNativePort,
			EtcdAddress:       "127.0.0.1",
			SecurePort:        secure,
			KubeAuthorization: authorization,
		})
		if err != nil {
			t.Fatal(err)
		}
		return component
	}
	controllerManager := func(t *testing.T, runtime string) internalversion.Component {
		component, err := BuildKubeControllerManagerComponent(BuildKubeControllerManagerComponentConfig{
			Runtime:           runtime,
			ProjectName:       "kwok-test",
			Version:           version.NewVersion(1, 37, 0),
			BindAddress:       "0.0.0.0",
			Port:              testNativePort,
			SecurePort:        true,
			KubeAuthorization: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return component
	}
	kwokController := func(t *testing.T, runtime string) internalversion.Component {
		return BuildKwokControllerComponent(BuildKwokControllerComponentConfig{
			Runtime:     runtime,
			ProjectName: "kwok-test",
			BindAddress: "0.0.0.0",
			Port:        testNativePort,
		})
	}
	etcd := func(t *testing.T, runtime string) internalversion.Component {
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
	}

	apiserverSecure := func(t *testing.T) internalversion.Component {
		return apiserver(t, consts.RuntimeTypeDocker, true, true)
	}
	apiserverInsecure := func(t *testing.T) internalversion.Component {
		return apiserver(t, consts.RuntimeTypeDocker, false, false)
	}

	tests := []struct {
		name      string
		component func(t *testing.T) internalversion.Component
		args      []string
		// wantPort is the port of the probe after the patch, which is never removed.
		wantPort uint32
		// wantConflict is true if the patched arguments keep the probe from succeeding.
		wantConflict bool
		// wantWarning is true if the probe is kept, but may not reach the patched address.
		wantWarning bool
	}{
		{name: "unpatched", component: apiserverSecure, wantPort: 6443},
		{name: "unrelated flag", component: apiserverSecure, args: []string{"--max-requests-inflight=100"}, wantPort: 6443},
		{name: "secure port moved", component: apiserverSecure, args: []string{"--secure-port=7443"}, wantPort: 7443},
		{
			name: "native secure port moved",
			component: func(t *testing.T) internalversion.Component {
				return apiserver(t, consts.RuntimeTypeBinary, true, true)
			},
			args:     []string{"--secure-port=7443"},
			wantPort: 7443,
		},
		{name: "secure port disabled", component: apiserverSecure, args: []string{"--secure-port=0"}, wantPort: 6443, wantConflict: true},
		{name: "invalid secure port", component: apiserverSecure, args: []string{"--secure-port=https"}, wantPort: 6443, wantConflict: true},
		{name: "anonymous authentication disabled", component: apiserverSecure, args: []string{"--anonymous-auth=false"}, wantPort: 6443, wantConflict: true},
		{name: "anonymous authentication disabled in capitals", component: apiserverSecure, args: []string{"--anonymous-auth=FALSE"}, wantPort: 6443, wantConflict: true},
		{name: "anonymous authentication disabled as a number", component: apiserverSecure, args: []string{"--anonymous-auth=0"}, wantPort: 6443, wantConflict: true},
		{name: "anonymous authentication enabled", component: apiserverSecure, args: []string{"--anonymous-auth"}, wantPort: 6443},
		{name: "anonymous authentication enabled in capitals", component: apiserverSecure, args: []string{"--anonymous-auth=True"}, wantPort: 6443},
		{name: "anonymous authentication enabled as a number", component: apiserverSecure, args: []string{"--anonymous-auth=1"}, wantPort: 6443},
		{name: "always allow authorizer", component: apiserverSecure, args: []string{"--authorization-mode=Node,AlwaysAllow"}, wantPort: 6443, wantConflict: true},
		{name: "another authorizer", component: apiserverSecure, args: []string{"--authorization-mode=Node,RBAC,Webhook"}, wantPort: 6443},
		{
			name: "bound to all interfaces",
			component: func(t *testing.T) internalversion.Component {
				return apiserver(t, consts.RuntimeTypeBinary, true, true)
			},
			args:     []string{"--bind-address=::"},
			wantPort: testNativePort,
		},
		{
			// The probe is kept and decides: the address may still be the one probed.
			name:        "bound to another address",
			component:   apiserverSecure,
			args:        []string{"--bind-address=127.0.0.1"},
			wantPort:    6443,
			wantWarning: true,
		},
		{
			// The insecure port is not probed in secure mode.
			name:      "insecure port in secure mode",
			component: apiserverSecure,
			args:      []string{"--insecure-port=0"},
			wantPort:  6443,
		},
		{name: "insecure port moved", component: apiserverInsecure, args: []string{"--insecure-port=9080"}, wantPort: 9080},
		{name: "insecure port disabled", component: apiserverInsecure, args: []string{"--insecure-port=0"}, wantPort: 8080, wantConflict: true},
		{
			name:        "insecure port bound to another address",
			component:   apiserverInsecure,
			args:        []string{"--insecure-bind-address=127.0.0.1"},
			wantPort:    8080,
			wantWarning: true,
		},
		{
			name:         "probed path no longer allowed",
			component:    func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:         []string{"--authorization-always-allow-paths=/metrics"},
			wantPort:     10257,
			wantConflict: true,
		},
		{
			name:      "probed path still allowed",
			component: func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:      []string{"--authorization-always-allow-paths=/healthz"},
			wantPort:  10257,
		},
		{
			name:      "probed path allowed by prefix",
			component: func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:      []string{"--authorization-always-allow-paths=/metrics,/health*"},
			wantPort:  10257,
		},
		{
			name:      "all paths allowed",
			component: func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:      []string{"--authorization-always-allow-paths=/*"},
			wantPort:  10257,
		},
		{
			// The leading "/" is optional.
			name:      "probed path allowed without the leading slash",
			component: func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:      []string{"--authorization-always-allow-paths=healthz"},
			wantPort:  10257,
		},
		{
			name:         "longer path allowed",
			component:    func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeDocker) },
			args:         []string{"--authorization-always-allow-paths=/healthz/ping"},
			wantPort:     10257,
			wantConflict: true,
		},
		{
			name:      "controller manager secure port moved",
			component: func(t *testing.T) internalversion.Component { return controllerManager(t, consts.RuntimeTypeBinary) },
			args:      []string{"--secure-port=20257"},
			wantPort:  20257,
		},
		{
			name:      "server address moved",
			component: func(t *testing.T) internalversion.Component { return kwokController(t, consts.RuntimeTypeDocker) },
			args:      []string{"--server-address=0.0.0.0:20247"},
			wantPort:  20247,
		},
		{
			name:        "server address on another address",
			component:   func(t *testing.T) internalversion.Component { return kwokController(t, consts.RuntimeTypeDocker) },
			args:        []string{"--server-address=127.0.0.1:10247"},
			wantPort:    10247,
			wantWarning: true,
		},
		{
			name:         "invalid server address",
			component:    func(t *testing.T) internalversion.Component { return kwokController(t, consts.RuntimeTypeDocker) },
			args:         []string{"--server-address=10247"},
			wantPort:     10247,
			wantConflict: true,
		},
		{
			name:      "client urls moved",
			component: func(t *testing.T) internalversion.Component { return etcd(t, consts.RuntimeTypeDocker) },
			args:      []string{"--listen-client-urls=https://0.0.0.0:2379,http://0.0.0.0:2381"},
			wantPort:  2381,
		},
		{
			name:         "client urls without plain http",
			component:    func(t *testing.T) internalversion.Component { return etcd(t, consts.RuntimeTypeDocker) },
			args:         []string{"--listen-client-urls=https://0.0.0.0:2379"},
			wantPort:     2379,
			wantConflict: true,
		},
		{
			name: "node-readiness-controller health address moved",
			component: func(t *testing.T) internalversion.Component {
				c, err := BuildNodeReadinessControllerComponent(BuildNodeReadinessControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 3, 0)})
				if err != nil {
					t.Fatal(err)
				}
				return c
			},
			args:     []string{"--health-probe-bind-address=:9081"},
			wantPort: 9081,
		},
		{
			// The flag overrides the configuration file of LWS.
			name: "lws health address moved",
			component: func(t *testing.T) internalversion.Component {
				c, err := BuildLWSComponent(BuildLWSComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 9, 0), HealthProbePort: 8081})
				if err != nil {
					t.Fatal(err)
				}
				return c
			},
			args:     []string{"--health-probe-bind-address=0.0.0.0:9081"},
			wantPort: 9081,
		},
		{
			name: "scheduler-plugins probe address moved",
			component: func(t *testing.T) internalversion.Component {
				c, err := BuildSchedulerPluginsControllerComponent(BuildSchedulerPluginsControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 34, 7)})
				if err != nil {
					t.Fatal(err)
				}
				return c
			},
			args:     []string{"--probeAddr=:9081"},
			wantPort: 9081,
		},
		{
			// "0" turns the endpoint off.
			name: "scheduler-plugins probe endpoint disabled",
			component: func(t *testing.T) internalversion.Component {
				c, err := BuildSchedulerPluginsControllerComponent(BuildSchedulerPluginsControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 34, 7)})
				if err != nil {
					t.Fatal(err)
				}
				return c
			},
			args:         []string{"--probeAddr=0"},
			wantPort:     8081,
			wantConflict: true,
		},
		{
			// The probe is kept on the same port and decides whether the address is reachable.
			name:      "client urls on another address",
			component: func(t *testing.T) internalversion.Component { return etcd(t, consts.RuntimeTypeDocker) },
			args:      []string{"--listen-client-urls=http://127.0.0.1:2379"},
			wantPort:  2379,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := tt.component(t)
			original := slices.Clone(component.Args)
			component.Args = append(component.Args, tt.args...)
			want := component.ReadinessProbe.DeepCopy()
			want.HTTPGet.Port = tt.wantPort

			err := ReconcileReadinessProbe(&component, original)
			if (err != nil) != (tt.wantConflict || tt.wantWarning) {
				t.Fatalf("ReconcileReadinessProbe() = %v, want an error %v", err, tt.wantConflict || tt.wantWarning)
			}
			// The probe is never removed, so that the component is never reported ready unprobed.
			if !reflect.DeepEqual(component.ReadinessProbe, want) {
				t.Fatalf("ReadinessProbe = %#v, want %#v", component.ReadinessProbe, want)
			}

			conflict := ReadinessProbeConflict(component)
			if (conflict != nil) != tt.wantConflict {
				t.Fatalf("ReadinessProbeConflict() = %v, want a conflict %v", conflict, tt.wantConflict)
			}
			if conflict != nil && !errors.Is(conflict, ErrReadinessProbeConflict) {
				t.Fatalf("ReadinessProbeConflict() = %v, want %v", conflict, ErrReadinessProbeConflict)
			}
		})
	}
}

func TestReconcileReadinessProbeWithoutProbe(t *testing.T) {
	// A component without a probe is not given one, and one with unknown flags is left as is.
	jobset := internalversion.Component{
		Name: consts.ComponentJobSet,
		Args: []string{"--config=/a"},
		ReadinessProbe: &internalversion.ComponentProbe{
			HTTPGet: &internalversion.ComponentHTTPGetAction{Path: "/readyz", Port: 8081},
		},
	}
	want := jobset.ReadinessProbe.DeepCopy()
	original := slices.Clone(jobset.Args)
	jobset.Args = append(jobset.Args, "--health-probe-bind-address=127.0.0.1:8081")
	if err := ReconcileReadinessProbe(&jobset, original); err != nil || !reflect.DeepEqual(jobset.ReadinessProbe, want) {
		t.Fatalf("ReconcileReadinessProbe() = %v, probe %#v, want %#v", err, jobset.ReadinessProbe, want)
	}

	scheduler := internalversion.Component{Name: consts.ComponentKubeScheduler, Args: []string{"--secure-port=10259"}}
	if err := ReconcileReadinessProbe(&scheduler, []string{"--secure-port=0"}); err != nil || scheduler.ReadinessProbe != nil {
		t.Fatalf("ReconcileReadinessProbe() = %v, probe %#v, want none", err, scheduler.ReadinessProbe)
	}
}

// Unpatched components must never conflict with their own probe, or they would never be ready.
func TestReadinessProbeConflictUnpatched(t *testing.T) {
	var built []internalversion.Component
	add := func(component internalversion.Component, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if component.ReadinessProbe == nil {
			t.Fatalf("%s has no readiness probe", component.Name)
		}
		built = append(built, component)
	}
	for _, runtime := range []string{consts.RuntimeTypeBinary, consts.RuntimeTypeDocker} {
		for _, mode := range []struct {
			version version.Version
			secure  bool
		}{
			{version.NewVersion(1, 37, 0), true},
			{version.NewVersion(1, 30, 0), true},
			{version.NewVersion(1, 13, 0), true},
			{version.NewVersion(1, 12, 10), false},
			{version.Unknown, true},
		} {
			add(BuildKubeApiserverComponent(BuildKubeApiserverComponentConfig{
				Runtime: runtime, ProjectName: "kwok-test", Version: mode.version, BindAddress: "0.0.0.0",
				Port: testNativePort, EtcdAddress: "127.0.0.1", SecurePort: mode.secure, KubeAuthorization: mode.secure,
			}))
			add(BuildKubeControllerManagerComponent(BuildKubeControllerManagerComponentConfig{
				Runtime: runtime, ProjectName: "kwok-test", Version: mode.version, BindAddress: "0.0.0.0",
				Port: testNativePort, SecurePort: mode.secure, KubeAuthorization: mode.secure,
			}))
			add(BuildKubeSchedulerComponent(BuildKubeSchedulerComponentConfig{
				Runtime: runtime, ProjectName: "kwok-test", Version: mode.version, BindAddress: "0.0.0.0",
				Port: testNativePort, SecurePort: mode.secure,
			}))
		}
		add(BuildEtcdComponent(BuildEtcdComponentConfig{
			Runtime: runtime, ProjectName: "kwok-test", Version: version.NewVersion(3, 6, 10), BindAddress: "127.0.0.1",
			Port: testNativePort, PeerPort: testNativePort + 1, QuotaBackendSize: "8Gi",
		}))
		add(BuildKwokControllerComponent(BuildKwokControllerComponentConfig{
			Runtime: runtime, ProjectName: "kwok-test", BindAddress: "0.0.0.0", Port: testNativePort,
		}), nil)
		add(BuildPrometheusComponent(BuildPrometheusComponentConfig{
			Runtime: runtime, BindAddress: "0.0.0.0", Port: testNativePort,
		}))
		add(BuildKubectlProxyComponent(BuildKubectlProxyComponentConfig{
			Runtime: runtime, ProjectName: "kwok-test", BindAddress: "0.0.0.0", Port: testNativePort,
		}))
	}
	add(BuildJobSetComponent(BuildJobSetComponentConfig{Runtime: consts.RuntimeTypeDocker, Image: "registry.k8s.io/jobset/jobset:v0.12.0"}))
	add(BuildKueueComponent(BuildKueueComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 18, 1), HealthProbePort: 8081}))
	add(BuildLWSComponent(BuildLWSComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 9, 0), HealthProbePort: 8081}))
	add(BuildNodeReadinessControllerComponent(BuildNodeReadinessControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 3, 0)}))
	add(BuildSchedulerPluginsControllerComponent(BuildSchedulerPluginsControllerComponentConfig{Runtime: consts.RuntimeTypeDocker, Version: version.NewVersion(0, 34, 7)}))

	for _, component := range built {
		if err := ReadinessProbeConflict(component); err != nil {
			t.Errorf("ReadinessProbeConflict(%s) = %v, want none", component.Name, err)
		}
	}
}

func TestAlwaysAllowed(t *testing.T) {
	tests := []struct {
		paths string
		path  string
		want  bool
	}{
		{"/healthz,/readyz,/livez,/metrics", "/healthz", true},
		{"/healthz,/readyz,/livez,/metrics", "/readyz", true},
		{"/metrics", "/healthz", false},
		{"healthz", "/healthz", true},
		{"/health*", "/healthz", true},
		{"/*", "/readyz", true},
		{"*", "/readyz", true},
		{"/healthz/*", "/healthz", false},
		{"/healthz/ping", "/healthz", false},
		{"", "/healthz", false},
	}
	for _, tt := range tests {
		if got := alwaysAllowed(tt.paths, tt.path); got != tt.want {
			t.Errorf("alwaysAllowed(%q, %q) = %v, want %v", tt.paths, tt.path, got, tt.want)
		}
	}
}
