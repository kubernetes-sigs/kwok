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

package runtime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func httpGetProbe(port uint32, path string) *internalversion.ComponentProbe {
	return &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path: path,
			Port: port,
		},
	}
}

func listenerPort(t *testing.T, addr net.Addr) uint32 {
	t.Helper()
	_, p, err := net.SplitHostPort(addr.String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(p, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(port)
}

func TestCheckReadinessProbeHTTPStatus(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/redirect":
			http.Redirect(w, r, "/elsewhere", http.StatusFound)
		case "/unavailable":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	port := listenerPort(t, server.Listener.Addr())

	tests := []struct {
		path    string
		wantErr bool
	}{
		{path: "/ok", wantErr: false},
		{path: "/redirect", wantErr: false},
		{path: "/missing", wantErr: true},
		{path: "/unavailable", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := CheckReadinessProbe(context.Background(), httpGetProbe(port, tt.path), "127.0.0.1", DialProbeRoundTrip("127.0.0.1"))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckReadinessProbe() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotPath != tt.path {
				t.Fatalf("server got path %q, want %q", gotPath, tt.path)
			}
		})
	}
}

func TestCheckReadinessProbeConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listenerPort(t, listener.Addr())
	_ = listener.Close()

	err = CheckReadinessProbe(context.Background(), httpGetProbe(port, "/readyz"), "127.0.0.1", DialProbeRoundTrip("127.0.0.1"))
	if err == nil {
		t.Fatal("expected error for refused connection")
	}
}

// hangingListener accepts connections and never responds.
func hangingListener(t *testing.T) uint32 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		_ = listener.Close()
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				<-done
				_ = conn.Close()
			}()
		}
	}()
	return listenerPort(t, listener.Addr())
}

func TestCheckReadinessProbeTimeout(t *testing.T) {
	port := hangingListener(t)
	probe := httpGetProbe(port, "/readyz")
	probe.TimeoutSeconds = 1

	start := time.Now()
	err := CheckReadinessProbe(context.Background(), probe, "127.0.0.1", DialProbeRoundTrip("127.0.0.1"))
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed < 900*time.Millisecond || elapsed > 5*time.Second {
		t.Fatalf("probe returned after %v, want about 1s", elapsed)
	}
}

func TestCheckReadinessProbeCanceledContext(t *testing.T) {
	port := hangingListener(t)
	probe := httpGetProbe(port, "/readyz")
	probe.TimeoutSeconds = 30

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	err := CheckReadinessProbe(ctx, probe, "127.0.0.1", DialProbeRoundTrip("127.0.0.1"))
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("probe returned after %v, want prompt return on cancel", elapsed)
	}

	// The cancellation reported by the transport is returned to the caller.
	called := false
	err = CheckReadinessProbe(ctx, probe, "127.0.0.1", func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		called = true
		return nil, ctx.Err()
	})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("CheckReadinessProbe() error = %v, want context.Canceled", err)
	}
	if !called {
		t.Fatal("transport should be called with the canceled context")
	}
}

func TestCheckReadinessProbeResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		rtErr    error
		wantErr  bool
	}{
		{
			name:     "complete response",
			response: "HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok",
		},
		{
			name:     "status line read before transport error",
			response: "HTTP/1.1 200 OK\r\n\r\n",
			rtErr:    errors.New("read timeout"),
		},
		{
			name:    "empty response",
			wantErr: true,
		},
		{
			name:    "transport error without response",
			rtErr:   errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:     "garbage response",
			response: "not http",
			wantErr:  true,
		},
		{
			name:     "informational status",
			response: "HTTP/1.1 101 Switching Protocols\r\n\r\n",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckReadinessProbe(context.Background(), httpGetProbe(8081, "/readyz"), "jobset",
				func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
					return []byte(tt.response), tt.rtErr
				})
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckReadinessProbe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckReadinessProbeRequest(t *testing.T) {
	var got string
	var gotPort uint32
	err := CheckReadinessProbe(context.Background(), httpGetProbe(8081, ""), "cluster-jobset",
		func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
			got = string(request)
			gotPort = port
			if _, ok := ctx.Deadline(); !ok {
				t.Error("transport context should have a deadline")
			}
			return []byte("HTTP/1.0 200 OK\r\n\r\n"), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if gotPort != 8081 {
		t.Fatalf("port = %d, want 8081", gotPort)
	}
	for _, want := range []string{
		"GET / HTTP/1.0\r\n",
		"Host: cluster-jobset:8081\r\n",
		"Connection: close\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("request %q does not contain %q", got, want)
		}
	}
	if !strings.HasSuffix(got, "\r\n\r\n") {
		t.Fatalf("request %q is not terminated", got)
	}
}

func TestCheckReadinessProbeInvalidConfig(t *testing.T) {
	tests := []struct {
		name  string
		probe *internalversion.ComponentProbe
	}{
		{name: "nil probe", probe: nil},
		{name: "no handler", probe: &internalversion.ComponentProbe{}},
		{name: "zero port", probe: httpGetProbe(0, "/readyz")},
		{name: "port out of range", probe: httpGetProbe(70000, "/readyz")},
		{name: "relative path", probe: httpGetProbe(8081, "readyz")},
		{name: "path with space", probe: httpGetProbe(8081, "/ready z")},
		{name: "path with newline", probe: httpGetProbe(8081, "/readyz\r\nX-Injected: 1")},
		{
			name: "negative timeout",
			probe: &internalversion.ComponentProbe{
				HTTPGet:        &internalversion.ComponentHTTPGetAction{Path: "/readyz", Port: 8081},
				TimeoutSeconds: -1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			err := CheckReadinessProbe(context.Background(), tt.probe, "127.0.0.1",
				func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
					called = true
					return []byte("HTTP/1.0 200 OK\r\n\r\n"), nil
				})
			if err == nil {
				t.Fatal("expected error for invalid probe")
			}
			if called {
				t.Fatal("transport should not be called for invalid probe")
			}
		})
	}
}

func TestComponentStatusWithProbe(t *testing.T) {
	ready := func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		return []byte("HTTP/1.0 200 OK\r\n\r\n"), nil
	}
	notReady := func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		return []byte("HTTP/1.0 503 Service Unavailable\r\n\r\n"), nil
	}
	unreachable := func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		return nil, errors.New("connection refused")
	}
	mustNotCall := func(t *testing.T) ProbeRoundTripFunc {
		return func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
			t.Error("transport should not be called")
			return nil, nil
		}
	}

	tests := []struct {
		name      string
		running   bool
		probe     *internalversion.ComponentProbe
		roundTrip func(t *testing.T) ProbeRoundTripFunc
		want      ComponentStatus
	}{
		{
			name:      "stopped without probe",
			running:   false,
			roundTrip: mustNotCall,
			want:      ComponentStatusStopped,
		},
		{
			name:      "stopped with probe",
			running:   false,
			probe:     httpGetProbe(8081, "/readyz"),
			roundTrip: mustNotCall,
			want:      ComponentStatusStopped,
		},
		{
			name:      "running without probe",
			running:   true,
			roundTrip: mustNotCall,
			want:      ComponentStatusReady,
		},
		{
			name:      "running with passing probe",
			running:   true,
			probe:     httpGetProbe(8081, "/readyz"),
			roundTrip: func(*testing.T) ProbeRoundTripFunc { return ready },
			want:      ComponentStatusReady,
		},
		{
			name:      "running with failing probe",
			running:   true,
			probe:     httpGetProbe(8081, "/readyz"),
			roundTrip: func(*testing.T) ProbeRoundTripFunc { return notReady },
			want:      ComponentStatusRunning,
		},
		{
			name:      "running with unreachable probe",
			running:   true,
			probe:     httpGetProbe(8081, "/readyz"),
			roundTrip: func(*testing.T) ProbeRoundTripFunc { return unreachable },
			want:      ComponentStatusRunning,
		},
		{
			name:      "running with invalid probe",
			running:   true,
			probe:     &internalversion.ComponentProbe{},
			roundTrip: mustNotCall,
			want:      ComponentStatusRunning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComponentStatusWithProbe(context.Background(), tt.running, tt.probe, "127.0.0.1", tt.roundTrip(t))
			if got != tt.want {
				t.Fatalf("ComponentStatusWithProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}
