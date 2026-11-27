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
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	goruntime "runtime"
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

func httpsGetProbe(port uint32, path string) *internalversion.ComponentProbe {
	probe := httpGetProbe(port, path)
	probe.HTTPGet.Scheme = internalversion.URISchemeHTTPS
	return probe
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

// checkNoGoroutineLeak fails the test if any goroutine is still running probe code at its end.
// Goroutines of the test servers are not counted, since they are stopped by later cleanups.
func checkNoGoroutineLeak(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			leaked := probeGoroutines()
			if len(leaked) == 0 {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("leaked goroutines:\n%s", strings.Join(leaked, "\n\n"))
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

func probeGoroutines() []string {
	buf := make([]byte, 1<<20)
	stacks := strings.Split(string(buf[:goruntime.Stack(buf, true)]), "\n\n")
	var leaked []string
	for _, stack := range stacks {
		if strings.Contains(stack, "kwokctl/runtime/probe.go") ||
			strings.Contains(stack, "crypto/tls.(*Conn).handshakeContext") {
			leaked = append(leaked, stack)
		}
	}
	return leaked
}

// respond returns a transport whose peer reads the request and replies with response, then closes the connection.
// A nil dialErr with an empty response closes the connection without replying.
func respond(response string, dialErr error) ProbeDialFunc {
	return func(ctx context.Context, port uint32) (net.Conn, error) {
		if dialErr != nil {
			return nil, dialErr
		}
		client, server := net.Pipe()
		go func() {
			defer func() {
				_ = server.Close()
			}()
			buf := make([]byte, 4096)
			var request []byte
			for !strings.Contains(string(request), "\r\n\r\n") {
				n, err := server.Read(buf)
				if err != nil {
					return
				}
				request = append(request, buf[:n]...)
			}
			_, _ = io.WriteString(server, response)
		}()
		return client, nil
	}
}

func newProbeHandler(gotPath *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotPath != nil {
			*gotPath = r.URL.Path
		}
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
	})
}

func TestCheckReadinessProbeHTTPStatus(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{path: "/ok", wantErr: false},
		{path: "/redirect", wantErr: false},
		{path: "/missing", wantErr: true},
		{path: "/unavailable", wantErr: true},
	}
	for _, scheme := range []internalversion.URIScheme{"", internalversion.URISchemeHTTP, internalversion.URISchemeHTTPS} {
		var gotPath string
		server := httptest.NewUnstartedServer(newProbeHandler(&gotPath))
		if scheme == internalversion.URISchemeHTTPS {
			// The test server certificate is self-signed.
			server.StartTLS()
		} else {
			server.Start()
		}
		port := listenerPort(t, server.Listener.Addr())

		for _, tt := range tests {
			t.Run(string(scheme)+tt.path, func(t *testing.T) {
				checkNoGoroutineLeak(t)
				probe := httpGetProbe(port, tt.path)
				probe.HTTPGet.Scheme = scheme
				err := CheckReadinessProbe(context.Background(), probe, "127.0.0.1", DialProbe("127.0.0.1"))
				if (err != nil) != tt.wantErr {
					t.Fatalf("CheckReadinessProbe() error = %v, wantErr %v", err, tt.wantErr)
				}
				if gotPath != tt.path {
					t.Fatalf("server got path %q, want %q", gotPath, tt.path)
				}
			})
		}
		server.Close()
	}
}

func TestCheckReadinessProbeSchemeMismatch(t *testing.T) {
	httpServer := httptest.NewServer(newProbeHandler(nil))
	t.Cleanup(httpServer.Close)
	httpsServer := httptest.NewTLSServer(newProbeHandler(nil))
	t.Cleanup(httpsServer.Close)

	t.Run("http probe of an https endpoint", func(t *testing.T) {
		checkNoGoroutineLeak(t)
		// Without a scheme, the probe keeps using HTTP.
		err := CheckReadinessProbe(context.Background(), httpGetProbe(listenerPort(t, httpsServer.Listener.Addr()), "/ok"),
			"127.0.0.1", DialProbe("127.0.0.1"))
		if err == nil {
			t.Fatal("expected error for an HTTP probe of an HTTPS endpoint")
		}
	})
	t.Run("https probe of an http endpoint", func(t *testing.T) {
		checkNoGoroutineLeak(t)
		err := CheckReadinessProbe(context.Background(), httpsGetProbe(listenerPort(t, httpServer.Listener.Addr()), "/ok"),
			"127.0.0.1", DialProbe("127.0.0.1"))
		if err == nil || !strings.Contains(err.Error(), "TLS handshake") {
			t.Fatalf("CheckReadinessProbe() error = %v, want a TLS handshake error", err)
		}
	})
	t.Run("https probe of a garbage endpoint", func(t *testing.T) {
		checkNoGoroutineLeak(t)
		err := CheckReadinessProbe(context.Background(), httpsGetProbe(8443, "/ok"), "127.0.0.1",
			respond("HTTP/1.0 200 OK\r\n\r\n", nil))
		if err == nil {
			t.Fatal("expected error for a plain HTTP response to an HTTPS probe")
		}
	})
}

func TestCheckReadinessProbeTLSConfig(t *testing.T) {
	var got *tls.ClientHelloInfo
	server := httptest.NewUnstartedServer(newProbeHandler(nil))
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			got = hello
			return nil, nil
		},
	}
	server.StartTLS()
	t.Cleanup(server.Close)
	port := listenerPort(t, server.Listener.Addr())

	// A host name is sent as the server name, an address is not.
	err := CheckReadinessProbe(context.Background(), httpsGetProbe(port, "/ok"), "localhost", DialProbe("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ServerName != "localhost" {
		t.Fatalf("server name = %v, want localhost", got)
	}
	err = CheckReadinessProbe(context.Background(), httpsGetProbe(port, "/ok"), "127.0.0.1", DialProbe("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	if got.ServerName != "" {
		t.Fatalf("server name = %q, want none for an address", got.ServerName)
	}
}

func TestCheckReadinessProbeConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listenerPort(t, listener.Addr())
	_ = listener.Close()

	for _, probe := range []*internalversion.ComponentProbe{httpGetProbe(port, "/readyz"), httpsGetProbe(port, "/readyz")} {
		err = CheckReadinessProbe(context.Background(), probe, "127.0.0.1", DialProbe("127.0.0.1"))
		if err == nil {
			t.Fatal("expected error for refused connection")
		}
	}
}

// hangingListener accepts connections, writes prefix, and never completes the response.
func hangingListener(t *testing.T, prefix func(conn net.Conn)) uint32 {
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
				if prefix != nil {
					prefix(conn)
				}
				<-done
				_ = conn.Close()
			}()
		}
	}()
	return listenerPort(t, listener.Addr())
}

// incompleteTLSResponse completes the TLS handshake and sends a status line, but never ends the header block.
func incompleteTLSResponse(t *testing.T) func(conn net.Conn) {
	server := httptest.NewUnstartedServer(nil)
	server.StartTLS()
	config := server.TLS.Clone()
	server.Close()
	return func(conn net.Conn) {
		tlsConn := tls.Server(conn, config)
		if tlsConn.Handshake() != nil {
			return
		}
		buf := make([]byte, 4096)
		_, _ = tlsConn.Read(buf)
		_, _ = io.WriteString(tlsConn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n")
	}
}

func TestCheckReadinessProbeTimeout(t *testing.T) {
	tests := []struct {
		name  string
		probe func(port uint32) *internalversion.ComponentProbe
		peer  func(t *testing.T) func(conn net.Conn)
	}{
		{
			name:  "http no response",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpGetProbe(port, "/readyz") },
		},
		{
			name: "http incomplete response",
			probe: func(port uint32) *internalversion.ComponentProbe {
				return httpGetProbe(port, "/readyz")
			},
			peer: func(*testing.T) func(conn net.Conn) {
				return func(conn net.Conn) {
					_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\n")
				}
			},
		},
		{
			name:  "https no handshake",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpsGetProbe(port, "/readyz") },
		},
		{
			name:  "https incomplete response",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpsGetProbe(port, "/readyz") },
			peer:  incompleteTLSResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var peer func(conn net.Conn)
			if tt.peer != nil {
				peer = tt.peer(t)
			}
			port := hangingListener(t, peer)
			checkNoGoroutineLeak(t)
			probe := tt.probe(port)
			probe.TimeoutSeconds = 1

			start := time.Now()
			err := CheckReadinessProbe(context.Background(), probe, "127.0.0.1", DialProbe("127.0.0.1"))
			elapsed := time.Since(start)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("CheckReadinessProbe() error = %v, want context.DeadlineExceeded", err)
			}
			if elapsed < 900*time.Millisecond || elapsed > 5*time.Second {
				t.Fatalf("probe returned after %v, want about 1s", elapsed)
			}
		})
	}
}

func TestCheckReadinessProbeCanceledContext(t *testing.T) {
	tests := []struct {
		name  string
		probe func(port uint32) *internalversion.ComponentProbe
		peer  func(t *testing.T) func(conn net.Conn)
	}{
		{
			name:  "http",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpGetProbe(port, "/readyz") },
		},
		{
			name:  "during tls handshake",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpsGetProbe(port, "/readyz") },
		},
		{
			name:  "during tls response",
			probe: func(port uint32) *internalversion.ComponentProbe { return httpsGetProbe(port, "/readyz") },
			peer:  incompleteTLSResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var peer func(conn net.Conn)
			if tt.peer != nil {
				peer = tt.peer(t)
			}
			port := hangingListener(t, peer)
			checkNoGoroutineLeak(t)
			probe := tt.probe(port)
			probe.TimeoutSeconds = 30

			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(100*time.Millisecond, cancel)

			start := time.Now()
			err := CheckReadinessProbe(ctx, probe, "127.0.0.1", DialProbe("127.0.0.1"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("CheckReadinessProbe() error = %v, want context.Canceled", err)
			}
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Fatalf("probe returned after %v, want prompt return on cancel", elapsed)
			}
		})
	}

	t.Run("reported by transport", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		err := CheckReadinessProbe(ctx, httpGetProbe(8081, "/readyz"), "127.0.0.1", func(ctx context.Context, port uint32) (net.Conn, error) {
			called = true
			return nil, ctx.Err()
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("CheckReadinessProbe() error = %v, want context.Canceled", err)
		}
		if !called {
			t.Fatal("transport should be called with the canceled context")
		}
	})
}

func TestCheckReadinessProbeResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		dialErr  error
		wantErr  bool
	}{
		{
			name:     "complete response",
			response: "HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok",
		},
		{
			name:     "response without body",
			response: "HTTP/1.1 200 OK\r\n\r\n",
		},
		{
			name:    "empty response",
			wantErr: true,
		},
		{
			name:    "transport error",
			dialErr: errors.New("connection refused"),
			wantErr: true,
		},
		{
			name:     "garbage response",
			response: "not http",
			wantErr:  true,
		},
		{
			name:     "truncated header block",
			response: "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n",
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
			checkNoGoroutineLeak(t)
			err := CheckReadinessProbe(context.Background(), httpGetProbe(8081, "/readyz"), "jobset", respond(tt.response, tt.dialErr))
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
		func(ctx context.Context, port uint32) (net.Conn, error) {
			gotPort = port
			if _, ok := ctx.Deadline(); !ok {
				t.Error("transport context should have a deadline")
			}
			client, server := net.Pipe()
			go func() {
				defer func() {
					_ = server.Close()
				}()
				buf := make([]byte, 4096)
				for !strings.HasSuffix(got, "\r\n\r\n") {
					n, err := server.Read(buf)
					if err != nil {
						return
					}
					got += string(buf[:n])
				}
				_, _ = io.WriteString(server, "HTTP/1.0 200 OK\r\n\r\n")
			}()
			return client, nil
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
}

func TestCheckReadinessProbeInvalidConfig(t *testing.T) {
	withScheme := func(scheme internalversion.URIScheme) *internalversion.ComponentProbe {
		probe := httpGetProbe(8081, "/readyz")
		probe.HTTPGet.Scheme = scheme
		return probe
	}
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
		// Schemes are case-sensitive, as they are for Pod probes.
		{name: "lowercase scheme", probe: withScheme("https")},
		{name: "unsupported scheme", probe: withScheme("TCP")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			err := CheckReadinessProbe(context.Background(), tt.probe, "127.0.0.1",
				func(ctx context.Context, port uint32) (net.Conn, error) {
					called = true
					return nil, errors.New("unexpected dial")
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
	ready := respond("HTTP/1.0 200 OK\r\n\r\n", nil)
	notReady := respond("HTTP/1.0 503 Service Unavailable\r\n\r\n", nil)
	unreachable := respond("", errors.New("connection refused"))
	mustNotCall := func(t *testing.T) ProbeDialFunc {
		return func(ctx context.Context, port uint32) (net.Conn, error) {
			t.Error("transport should not be called")
			return nil, errors.New("unexpected dial")
		}
	}

	tests := []struct {
		name    string
		running bool
		probe   *internalversion.ComponentProbe
		dial    func(t *testing.T) ProbeDialFunc
		want    ComponentStatus
	}{
		{
			name:    "stopped without probe",
			running: false,
			dial:    mustNotCall,
			want:    ComponentStatusStopped,
		},
		{
			name:    "stopped with probe",
			running: false,
			probe:   httpGetProbe(8081, "/readyz"),
			dial:    mustNotCall,
			want:    ComponentStatusStopped,
		},
		{
			name:    "running without probe",
			running: true,
			dial:    mustNotCall,
			want:    ComponentStatusReady,
		},
		{
			name:    "running with passing probe",
			running: true,
			probe:   httpGetProbe(8081, "/readyz"),
			dial:    func(*testing.T) ProbeDialFunc { return ready },
			want:    ComponentStatusReady,
		},
		{
			name:    "running with failing probe",
			running: true,
			probe:   httpGetProbe(8081, "/readyz"),
			dial:    func(*testing.T) ProbeDialFunc { return notReady },
			want:    ComponentStatusRunning,
		},
		{
			name:    "running with unreachable probe",
			running: true,
			probe:   httpGetProbe(8081, "/readyz"),
			dial:    func(*testing.T) ProbeDialFunc { return unreachable },
			want:    ComponentStatusRunning,
		},
		{
			name:    "running with invalid probe",
			running: true,
			probe:   &internalversion.ComponentProbe{},
			dial:    mustNotCall,
			want:    ComponentStatusRunning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComponentStatusWithProbe(context.Background(), tt.running, tt.probe, "127.0.0.1", tt.dial(t))
			if got != tt.want {
				t.Fatalf("ComponentStatusWithProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}

// serveRaw starts a listener whose connections read the request and then run write.
func serveRaw(t *testing.T, write func(conn net.Conn)) uint32 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() {
					_ = conn.Close()
				}()
				buf := make([]byte, 4096)
				_, _ = conn.Read(buf)
				write(conn)
			}()
		}
	}()
	return listenerPort(t, listener.Addr())
}

func TestCheckReadinessProbeOversizedResponse(t *testing.T) {
	tests := []struct {
		name    string
		write   func(conn net.Conn)
		wantErr bool
	}{
		{
			// The header block is read up to a limit, rather than until the probe times out.
			name: "endless headers",
			write: func(conn net.Conn) {
				_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\n")
				for {
					if _, err := io.WriteString(conn, "X-Padding: "+strings.Repeat("a", 1000)+"\r\n"); err != nil {
						return
					}
				}
			},
			wantErr: true,
		},
		{
			name: "oversized header line",
			write: func(conn net.Conn) {
				_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nX-Padding: "+strings.Repeat("a", 2*maxProbeResponse)+"\r\n\r\n")
			},
			wantErr: true,
		},
		{
			// The body is never read.
			name: "endless body",
			write: func(conn net.Conn) {
				_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\n\r\n")
				for {
					if _, err := io.WriteString(conn, strings.Repeat("b", 4096)); err != nil {
						return
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := serveRaw(t, tt.write)
			checkNoGoroutineLeak(t)
			probe := httpGetProbe(port, "/readyz")
			probe.TimeoutSeconds = 10

			start := time.Now()
			err := CheckReadinessProbe(context.Background(), probe, "127.0.0.1", DialProbe("127.0.0.1"))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckReadinessProbe() error = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("CheckReadinessProbe() error = %v, want it to end before the timeout", err)
			}
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Fatalf("probe returned after %v, want it to end before the timeout", elapsed)
			}
		})
	}
}

func TestCheckReadinessProbeSendsNoCredentials(t *testing.T) {
	type request struct {
		certificates int
		headers      http.Header
	}
	got := make(chan request, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- request{certificates: len(r.TLS.PeerCertificates), headers: r.Header.Clone()}
	}))
	// The server asks for a client certificate, which the probe must not present.
	server.TLS = &tls.Config{ClientAuth: tls.RequestClientCert, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	t.Cleanup(server.Close)

	err := CheckReadinessProbe(context.Background(), httpsGetProbe(listenerPort(t, server.Listener.Addr()), "/readyz"),
		"127.0.0.1", DialProbe("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	r := <-got
	if r.certificates != 0 {
		t.Errorf("client certificates = %d, want none", r.certificates)
	}
	for _, header := range []string{"Authorization", "Cookie", "Proxy-Authorization"} {
		if v := r.headers.Get(header); v != "" {
			t.Errorf("%s = %q, want none", header, v)
		}
	}
}
