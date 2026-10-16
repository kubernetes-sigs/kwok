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

package binary

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// newProbeTarget starts an HTTP server on address whose /healthz succeeds only while ready is true.
func newProbeTarget(t *testing.T, address string, ready *atomic.Bool) uint32 {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(address, "0"))
	if err != nil {
		t.Skipf("cannot listen on %s: %v", address, err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/healthz" || !ready.Load() {
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = rw.Write([]byte("ok"))
	}))
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return uint32(listener.Addr().(*net.TCPAddr).Port)
}

// newTestCluster returns a cluster with a single component that is running as the current test process.
func newTestCluster(t *testing.T, bindAddress string, probe *internalversion.ComponentProbe) *Cluster {
	t.Helper()
	workdir := t.TempDir()
	binary := utilspath.Join(workdir, "bin", "fake-component")

	pidPath := utilspath.Join(workdir, "pids", "fake-component.pid")
	if err := os.MkdirAll(utilspath.Dir(pidPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o640); err != nil {
		t.Fatal(err)
	}

	c := &Cluster{
		Cluster: runtime.NewCluster("test", workdir),
	}
	err := c.SetConfig(context.Background(), &internalversion.KwokctlConfiguration{
		Options: internalversion.KwokctlConfigurationOptions{
			BindAddress: bindAddress,
		},
		Components: []internalversion.Component{
			{
				Name:           "fake-component",
				Binary:         binary,
				WorkDir:        workdir,
				ReadinessProbe: probe,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func inspect(t *testing.T, c *Cluster) runtime.ComponentStatus {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status, err := c.InspectComponent(ctx, "fake-component")
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func TestInspectComponentReadinessProbe(t *testing.T) {
	tests := []struct {
		name        string
		bindAddress string
		listen      string
	}{
		{
			name:        "all interfaces",
			bindAddress: "0.0.0.0",
			listen:      "127.0.0.1",
		},
		{
			// Not reachable over 127.0.0.1, so the configured address must be used.
			name:        "specific address",
			bindAddress: "127.0.0.2",
			listen:      "127.0.0.2",
		},
		{
			name:        "all ipv6 interfaces",
			bindAddress: "::",
			listen:      "::1",
		},
		{
			name:        "specific ipv6 address",
			bindAddress: "::1",
			listen:      "::1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ready atomic.Bool
			port := newProbeTarget(t, tt.listen, &ready)
			c := newTestCluster(t, tt.bindAddress, &internalversion.ComponentProbe{
				HTTPGet: &internalversion.ComponentHTTPGetAction{
					Path: "/healthz",
					Port: port,
				},
			})

			if got := inspect(t, c); got != runtime.ComponentStatusRunning {
				t.Fatalf("status with a failing endpoint = %v, want %v", got, runtime.ComponentStatusRunning)
			}

			ready.Store(true)
			if got := inspect(t, c); got != runtime.ComponentStatusReady {
				t.Fatalf("status with a succeeding endpoint = %v, want %v", got, runtime.ComponentStatusReady)
			}

			ready.Store(false)
			if got := inspect(t, c); got != runtime.ComponentStatusRunning {
				t.Fatalf("status after the endpoint fails again = %v, want %v", got, runtime.ComponentStatusRunning)
			}
		})
	}
}

func TestInspectComponentReadinessProbeNotListening(t *testing.T) {
	var ready atomic.Bool
	ready.Store(true)
	// The component listens on another address than the one kwokctl was configured with.
	port := newProbeTarget(t, "127.0.0.1", &ready)
	c := newTestCluster(t, "192.0.2.1", &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path: "/healthz",
			Port: port,
		},
		TimeoutSeconds: 1,
	})

	if got := inspect(t, c); got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, want %v", got, runtime.ComponentStatusRunning)
	}
}

func TestInspectComponentWithoutReadinessProbe(t *testing.T) {
	c := newTestCluster(t, "0.0.0.0", nil)

	if got := inspect(t, c); got != runtime.ComponentStatusReady {
		t.Fatalf("status = %v, want %v", got, runtime.ComponentStatusReady)
	}

	if err := os.Remove(utilspath.Join(c.Workdir(), "pids", "fake-component.pid")); err != nil {
		t.Fatal(err)
	}
	if got := inspect(t, c); got != runtime.ComponentStatusStopped {
		t.Fatalf("status after stop = %v, want %v", got, runtime.ComponentStatusStopped)
	}
}

func TestProbeHost(t *testing.T) {
	tests := []struct {
		bindAddress string
		want        string
	}{
		{"", "127.0.0.1"},
		{"0.0.0.0", "127.0.0.1"},
		{"::", "::1"},
		{"127.0.0.1", "127.0.0.1"},
		{"192.0.2.1", "192.0.2.1"},
		{"::1", "::1"},
		{"localhost", "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.bindAddress, func(t *testing.T) {
			if got := probeHost(tt.bindAddress); got != tt.want {
				t.Errorf("probeHost(%q) = %q, want %q", tt.bindAddress, got, tt.want)
			}
		})
	}
}
