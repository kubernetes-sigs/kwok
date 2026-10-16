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

package compose

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// fakeRuntime is a container runtime CLI that answers inspect and the nc relay from state files.
// <container>.running or <container>.stopped sets the container state,
// <container>.ip its address on the cluster network named in the network file,
// and <container>.ready makes the relayed probe of that container succeed.
// The relay behaves like the real one: a Go server cancels a request whose client half-closed,
// which nc does at the end of its input, and nc only exits once its input ends.
// Like the musl resolver of the kwok-controller image, nc cannot resolve a label longer than 63 characters.
// With <container>.relay, nc instead relays bytes in both directions to the same port on 127.0.0.1,
// so that a test server can answer, over TLS or not.
// The PID of every process started by the relay is appended to the pids file.
const fakeRuntime = `#!/usr/bin/env bash
state="$(dirname "${0}")"
case "${1}" in
inspect)
  if [[ -f "${state}/${2}.running" && -f "${state}/${2}.ip" ]]; then
    echo '{"State":{"Running":true},"NetworkSettings":{"Networks":{"'"$(cat "${state}/network")"'":{"IPAddress":"'"$(cat "${state}/${2}.ip")"'"}}}}'
  elif [[ -f "${state}/${2}.running" ]]; then
    echo '{"State":{"Running":true}}'
  elif [[ -f "${state}/${2}.stopped" ]]; then
    echo '{"State":{"Running":false}}'
  else
    echo "Error: No such object: ${2}" >&2
    exit 1
  fi
  ;;
exec)
  # exec -i <container> nc -w <timeout> <host> <port>
  if [[ ! -f "${state}/${3}.running" ]]; then
    echo "Error response from daemon: container ${3} is not running" >&2
    exit 1
  fi
  echo "${3} ${4} ${7}:${8}" >>"${state}/requests"
  if [[ "${7}" =~ [^.]{64} ]]; then
    echo "nc: bad address '${7}'" >&2
    exit 1
  fi
  target="${7}"
  for ip in "${state}"/*.ip; do
    if [[ -f "${ip}" && "$(cat "${ip}")" == "${7}" ]]; then
      target="$(basename "${ip}" .ip)"
    fi
  done
  echo "${$}" >>"${state}/pids"
  if [[ -f "${state}/${target}.relay" ]]; then
    exec 3<>"/dev/tcp/127.0.0.1/${8}" || exit 1
    # An asynchronous command reads /dev/null unless its input is redirected.
    cat <&0 >&3 2>/dev/null &
    echo "${!}" >>"${state}/pids"
    # Only the output is relayed by this process, so killing it ends the output.
    exec cat <&3
  fi
  while IFS= read -r line && [[ "${line}" != $'\r' ]]; do :; done
  if read -r -t 1 -n 1 _ || [[ "${?}" -gt 128 ]]; then
    if [[ -f "${state}/${target}.ready" ]]; then
      printf 'HTTP/1.0 200 OK\r\nContent-Length: 2\r\n\r\nok'
    else
      printf 'HTTP/1.0 503 Service Unavailable\r\nContent-Length: 0\r\n\r\n'
    fi
  else
    printf 'HTTP/1.0 500 Internal Server Error\r\nContent-Length: 16\r\n\r\ncontext canceled'
  fi
  cat >/dev/null
  ;;
*)
  exit 2
  ;;
esac
`

type fakeCompose struct {
	*Cluster
	state string
}

func newFakeCompose(t *testing.T, components ...internalversion.Component) *fakeCompose {
	t.Helper()
	return newFakeComposeWithName(t, "test", components...)
}

func newFakeComposeWithName(t *testing.T, name string, components ...internalversion.Component) *fakeCompose {
	t.Helper()
	if goruntime.GOOS == "windows" {
		t.Skip("the fake container runtime is a bash script")
	}

	state := t.TempDir()
	bin := utilspath.Join(state, "fake-runtime")
	if err := os.WriteFile(bin, []byte(fakeRuntime), 0o750); err != nil {
		t.Fatal(err)
	}

	c := &Cluster{
		Cluster: runtime.NewCluster(name, t.TempDir()),
		runtime: bin,
	}
	// The cluster network is named after the cluster.
	if err := os.WriteFile(utilspath.Join(state, "network"), []byte(c.Name()), 0o640); err != nil {
		t.Fatal(err)
	}
	err := c.SetConfig(context.Background(), &internalversion.KwokctlConfiguration{
		Components: components,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &fakeCompose{Cluster: c, state: state}
}

// set replaces the state of a container with the given state, or removes it if state is empty.
func (f *fakeCompose) set(t *testing.T, component string, states ...string) {
	t.Helper()
	container := f.Name() + "-" + component
	for _, s := range []string{"running", "stopped", "ready", "ip", "relay"} {
		_ = os.Remove(utilspath.Join(f.state, container+"."+s))
	}
	for _, s := range states {
		if err := os.WriteFile(utilspath.Join(f.state, container+"."+s), nil, 0o640); err != nil {
			t.Fatal(err)
		}
	}
}

// setIP sets the address of a container on the cluster network.
func (f *fakeCompose) setIP(t *testing.T, component, ip string) {
	t.Helper()
	if err := os.WriteFile(utilspath.Join(f.state, f.Name()+"-"+component+".ip"), []byte(ip), 0o640); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeCompose) requests(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(utilspath.Join(f.state, "requests"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func (f *fakeCompose) inspect(t *testing.T, name string) (runtime.ComponentStatus, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return f.InspectComponent(ctx, name)
}

var (
	testJobSet = internalversion.Component{
		Name: consts.ComponentJobSet,
		ReadinessProbe: &internalversion.ComponentProbe{
			HTTPGet: &internalversion.ComponentHTTPGetAction{
				Path: "/readyz",
				Port: 8081,
			},
			TimeoutSeconds: 5,
		},
	}
	testKwokController = internalversion.Component{
		Name: consts.ComponentKwokController,
	}
	testKubeScheduler = internalversion.Component{
		Name: consts.ComponentKubeScheduler,
	}
)

func TestInspectComponentProbeRelayedByKwokController(t *testing.T) {
	f := newFakeCompose(t, testKwokController, testJobSet)
	f.set(t, consts.ComponentKwokController, "running")

	f.set(t, consts.ComponentJobSet, "running")
	got, err := f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status with a failing probe = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}

	f.set(t, consts.ComponentJobSet, "running", "ready")
	start := time.Now()
	got, err = f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status with a succeeding probe = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
	// The relay ends once the response headers arrive, rather than when the probe times out.
	// The fake waits up to a second to detect a half-close, so allow for that.
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("probe took %v, want it to end once the response arrives", elapsed)
	}

	// Both checks are relayed by nc in the kwok-controller container to the jobset container.
	request := f.Name() + "-kwok-controller nc " + f.Name() + "-jobset:8081"
	want := []string{request, request}
	if got := f.requests(t); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("relayed requests = %q, want %q", got, want)
	}

	f.set(t, consts.ComponentJobSet, "stopped")
	got, err = f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusStopped {
		t.Fatalf("status of a stopped component = %v, %v, want %v", got, err, runtime.ComponentStatusStopped)
	}
}

func TestInspectComponentProbeKwokControllerStopped(t *testing.T) {
	f := newFakeCompose(t, testKwokController, testJobSet)
	f.set(t, consts.ComponentKwokController, "stopped")
	// Even an endpoint that would succeed is not ready if it cannot be probed.
	f.set(t, consts.ComponentJobSet, "running", "ready")

	got, err := f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}
	if got := f.requests(t); len(got) != 0 {
		t.Errorf("relayed requests = %q, want none", got)
	}
}

func TestInspectComponentProbeKwokControllerDisabled(t *testing.T) {
	f := newFakeCompose(t, testJobSet, testKubeScheduler)
	f.set(t, consts.ComponentJobSet, "running", "ready")
	f.set(t, consts.ComponentKubeScheduler, "running")

	got, err := f.inspect(t, consts.ComponentJobSet)
	if !errors.Is(err, errProbeRelayUnavailable) || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v, %v", got, err, runtime.ComponentStatusRunning, errProbeRelayUnavailable)
	}

	ready, err := f.Ready(context.Background())
	if ready {
		t.Fatalf("Ready() = %v, %v, want false", ready, err)
	}

	// A component without a probe keeps the previous behavior.
	got, err = f.inspect(t, consts.ComponentKubeScheduler)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status without a probe = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
}

var testKubectlProxy = internalversion.Component{
	Name: consts.ComponentKubeApiserverInsecureProxy,
	ReadinessProbe: &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path: "/healthz",
			Port: 8001,
		},
		TimeoutSeconds: 5,
	},
}

func TestInspectComponentProbeLongClusterName(t *testing.T) {
	// The container name of the proxy is 66 characters, longer than a DNS label can be.
	f := newFakeComposeWithName(t, "kwokctl-readiness-docker-1-37-0", testKwokController, testKubectlProxy)
	container := f.Name() + "-" + consts.ComponentKubeApiserverInsecureProxy
	if len(container) <= 63 {
		t.Fatalf("container name %q must be longer than a DNS label", container)
	}
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserverInsecureProxy, "running", "ready")
	f.setIP(t, consts.ComponentKubeApiserverInsecureProxy, "172.18.0.9")

	got, err := f.inspect(t, consts.ComponentKubeApiserverInsecureProxy)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
	want := []string{f.Name() + "-kwok-controller nc 172.18.0.9:8001"}
	if got := f.requests(t); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("relayed requests = %q, want %q", got, want)
	}
}

func TestInspectComponentProbeWithoutAddress(t *testing.T) {
	// Without an address on the cluster network, a container name too long to resolve is not relayed,
	// and the probe fails closed with an error that says why.
	f := newFakeComposeWithName(t, "kwokctl-readiness-docker-1-37-0", testKwokController, testKubectlProxy)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserverInsecureProxy, "running", "ready")

	got, err := f.inspect(t, consts.ComponentKubeApiserverInsecureProxy)
	if !errors.Is(err, errProbeAddressUnavailable) || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v, %v", got, err, runtime.ComponentStatusRunning, errProbeAddressUnavailable)
	}
	if got := f.requests(t); len(got) != 0 {
		t.Errorf("relayed requests = %q, want none", got)
	}

	ready, err := f.Ready(context.Background())
	if ready {
		t.Fatalf("Ready() = %v, %v, want false", ready, err)
	}
}

// setRelay makes nc relay to a test server listening on 127.0.0.1 at the probed port.
func (f *fakeCompose) setRelay(t *testing.T, component string) {
	t.Helper()
	if err := os.WriteFile(utilspath.Join(f.state, f.Name()+"-"+component+".relay"), nil, 0o640); err != nil {
		t.Fatal(err)
	}
}

// checkNoLeak fails the test if a relay process or a relay goroutine is still running at its end.
func (f *fakeCompose) checkNoLeak(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for {
			leaked := f.leaked(t)
			if len(leaked) == 0 {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("leaked: %q", leaked)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

func (f *fakeCompose) leaked(t *testing.T) []string {
	t.Helper()
	var leaked []string
	data, err := os.ReadFile(utilspath.Join(f.state, "pids"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	for pid := range strings.FieldsSeq(string(data)) {
		// A zombie is gone, since it holds no resources but its exit status.
		stat, err := os.ReadFile("/proc/" + pid + "/stat")
		if err == nil && !strings.Contains(string(stat), ") Z ") {
			leaked = append(leaked, "process "+pid)
		}
	}
	buf := make([]byte, 1<<20)
	for stack := range strings.SplitSeq(string(buf[:goruntime.Stack(buf, true)]), "\n\n") {
		if strings.Contains(stack, "compose.(*Cluster).probeDial") ||
			strings.Contains(stack, "compose.(*probeRelayConn)") ||
			strings.Contains(stack, "kwokctl/runtime/probe.go") {
			leaked = append(leaked, stack)
		}
	}
	return leaked
}

// newTLSProbeTarget starts an HTTPS server with a self-signed certificate whose /readyz succeeds only while ready is true.
func newTLSProbeTarget(t *testing.T, ready *atomic.Bool) uint32 {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/readyz" || !ready.Load() {
			rw.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = rw.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	return uint32(server.Listener.Addr().(*net.TCPAddr).Port)
}

func probeComponent(port uint32, scheme internalversion.URIScheme, timeoutSeconds int32) internalversion.Component {
	return internalversion.Component{
		Name: consts.ComponentKubeApiserver,
		ReadinessProbe: &internalversion.ComponentProbe{
			HTTPGet: &internalversion.ComponentHTTPGetAction{
				Path:   "/readyz",
				Port:   port,
				Scheme: scheme,
			},
			TimeoutSeconds: timeoutSeconds,
		},
	}
}

func requireOnLinux(t *testing.T) {
	t.Helper()
	if goruntime.GOOS != "linux" {
		t.Skip("the relay processes are checked through /proc")
	}
}

func TestInspectComponentHTTPSProbeRelayed(t *testing.T) {
	requireOnLinux(t)
	var ready atomic.Bool
	port := newTLSProbeTarget(t, &ready)

	f := newFakeComposeWithName(t, "kwokctl-readiness-docker-1-37-0", testKwokController,
		probeComponent(port, internalversion.URISchemeHTTPS, 5))
	f.checkNoLeak(t)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserver, "running")
	f.setIP(t, consts.ComponentKubeApiserver, "172.18.0.3")
	f.setRelay(t, consts.ComponentKubeApiserver)

	got, err := f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status with a failing probe = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}

	ready.Store(true)
	start := time.Now()
	got, err = f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status with a succeeding probe = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
	// The relay ends with the response, rather than when the probe times out.
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("probe took %v, want it to end once the response arrives", elapsed)
	}

	// TLS runs over nc, which only connects to the address of the component on the cluster network.
	request := f.Name() + "-kwok-controller nc 172.18.0.3:" + strconv.FormatUint(uint64(port), 10)
	want := []string{request, request}
	if got := f.requests(t); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("relayed requests = %q, want %q", got, want)
	}
}

func TestInspectComponentHTTPProbeOfHTTPSEndpointRelayed(t *testing.T) {
	requireOnLinux(t)
	var ready atomic.Bool
	ready.Store(true)
	port := newTLSProbeTarget(t, &ready)

	// A probe without a scheme keeps using HTTP, which an HTTPS endpoint rejects.
	f := newFakeCompose(t, testKwokController, probeComponent(port, "", 5))
	f.checkNoLeak(t)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserver, "running")
	f.setRelay(t, consts.ComponentKubeApiserver)

	got, err := f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}
}

func TestInspectComponentHTTPSProbeRelayEOF(t *testing.T) {
	requireOnLinux(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint32(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()

	tests := []struct {
		name  string
		setup func(f *fakeCompose)
	}{
		{
			// nc fails to connect and exits without output.
			name: "connection refused",
			setup: func(f *fakeCompose) {
				f.set(t, consts.ComponentKwokController, "running")
				f.set(t, consts.ComponentKubeApiserver, "running")
				f.setRelay(t, consts.ComponentKubeApiserver)
			},
		},
		{
			// The exec fails before nc starts.
			name: "relay container stopped",
			setup: func(f *fakeCompose) {
				f.set(t, consts.ComponentKwokController, "stopped")
				f.set(t, consts.ComponentKubeApiserver, "running")
				f.setRelay(t, consts.ComponentKubeApiserver)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeCompose(t, testKwokController, probeComponent(port, internalversion.URISchemeHTTPS, 30))
			f.checkNoLeak(t)
			tt.setup(f)

			start := time.Now()
			got, err := f.inspect(t, consts.ComponentKubeApiserver)
			if err != nil || got != runtime.ComponentStatusRunning {
				t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
			}
			// The end of the relay output fails the probe at once, rather than when it times out.
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Errorf("probe took %v, want it to end with the relay", elapsed)
			}
		})
	}
}

func TestInspectComponentHTTPSProbeRelayTimeout(t *testing.T) {
	requireOnLinux(t)
	// The server accepts the connection and never answers the TLS handshake.
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
	port := uint32(listener.Addr().(*net.TCPAddr).Port)

	f := newFakeCompose(t, testKwokController, probeComponent(port, internalversion.URISchemeHTTPS, 1))
	f.checkNoLeak(t)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserver, "running")
	f.setRelay(t, consts.ComponentKubeApiserver)

	start := time.Now()
	got, err := f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}
	// The relay is killed once the probe times out, even though its peer keeps the connection open.
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond || elapsed > 4*time.Second {
		t.Errorf("probe took %v, want about the probe timeout", elapsed)
	}
}

func TestInspectComponentProbeRelayPeerKeepsConnection(t *testing.T) {
	requireOnLinux(t)
	// The peer answers and keeps the connection open, so nc does not exit on its own.
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
				buf := make([]byte, 4096)
				_, _ = conn.Read(buf)
				_, _ = io.WriteString(conn, "HTTP/1.0 200 OK\r\n\r\n")
				<-done
				_ = conn.Close()
			}()
		}
	}()
	port := uint32(listener.Addr().(*net.TCPAddr).Port)

	f := newFakeCompose(t, testKwokController, probeComponent(port, internalversion.URISchemeHTTP, 15))
	f.checkNoLeak(t)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserver, "running")
	f.setRelay(t, consts.ComponentKubeApiserver)

	start := time.Now()
	got, err := f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
	// The relay is killed shortly after the response, rather than when the probe times out.
	if elapsed := time.Since(start); elapsed > probeRelayCloseGrace+3*time.Second {
		t.Errorf("probe took %v, want it to end after the close grace", elapsed)
	}
}

func TestInspectComponentPatchedProbeConflict(t *testing.T) {
	// The endpoint would answer, but the patched arguments keep the real one from answering the probe.
	apiserver := probeComponent(6443, internalversion.URISchemeHTTPS, 5)
	apiserver.Args = []string{"--secure-port=6443", "--authorization-mode=AlwaysAllow"}
	f := newFakeCompose(t, testKwokController, apiserver)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserver, "running", "ready")
	f.setIP(t, consts.ComponentKubeApiserver, "172.18.0.3")

	got, err := f.inspect(t, consts.ComponentKubeApiserver)
	if !errors.Is(err, components.ErrReadinessProbeConflict) || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v, %v", got, err, runtime.ComponentStatusRunning, components.ErrReadinessProbeConflict)
	}
	if got := f.requests(t); len(got) != 0 {
		t.Errorf("relayed requests = %q, want none", got)
	}
	ready, err := f.Ready(context.Background())
	if ready {
		t.Fatalf("Ready() = %v, %v, want false", ready, err)
	}

	f.set(t, consts.ComponentKubeApiserver, "stopped")
	got, err = f.inspect(t, consts.ComponentKubeApiserver)
	if err != nil || got != runtime.ComponentStatusStopped {
		t.Fatalf("status of a stopped component = %v, %v, want %v", got, err, runtime.ComponentStatusStopped)
	}
}

func TestInspectComponentJobSetWebhookRegistration(t *testing.T) {
	requireOnLinux(t)
	caCertPath := utilspath.Join(t.TempDir(), "ca.crt")
	if err := os.WriteFile(caCertPath, []byte("ca"), 0o600); err != nil {
		t.Fatal(err)
	}
	jobset, err := components.BuildJobSetComponent(components.BuildJobSetComponentConfig{
		Runtime:     consts.RuntimeTypeDocker,
		ProjectName: "kwok-test",
		CaCertPath:  caCertPath,
		RawManifests: []string{`apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: jobset-validating-webhook-configuration
webhooks:
- name: vpod.kb.io
  clientConfig:
    service:
      name: jobset-webhook-service
      namespace: jobset-system
      path: /validate--v1-pod
`},
	})
	if err != nil {
		t.Fatal(err)
	}
	probe := jobset.ReadinessProbe.HTTPGet
	if probe.Scheme != internalversion.URISchemeHTTPS || probe.Port != 9443 || probe.Path != "/validate--v1-pod" {
		t.Fatalf("ReadinessProbe = %#v, want an HTTPS GET of /validate--v1-pod on 9443", probe)
	}

	// The webhook server listens before the webhook is registered, so the path is not found at first,
	// as it is for jobset between the start of its webhook server and the registration of its webhooks.
	mux := http.NewServeMux()
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	// The relay connects to the probed port on 127.0.0.1, where 9443 may be taken, so the test server's port is probed.
	probe.Port = uint32(server.Listener.Addr().(*net.TCPAddr).Port)

	f := newFakeCompose(t, testKwokController, jobset)
	f.checkNoLeak(t)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentJobSet, "running")
	f.setIP(t, consts.ComponentJobSet, "172.18.0.5")
	f.setRelay(t, consts.ComponentJobSet)

	got, err := f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status before the webhook is registered = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}

	// Like a controller-runtime webhook, the registered handler answers a GET without a body with 200
	// and an AdmissionReview that rejects it. So 200 shows that the webhook is registered and served,
	// not that it admits anything.
	mux.HandleFunc(probe.Path, func(rw http.ResponseWriter, req *http.Request) {
		if req.ContentLength > 0 {
			t.Errorf("probe sent a body of %d bytes, want none", req.ContentLength)
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(rw, `{"response":{"uid":"","allowed":false,"status":{"message":"request body is empty","code":400}}}`)
	})

	got, err = f.inspect(t, consts.ComponentJobSet)
	if err != nil || got != runtime.ComponentStatusReady {
		t.Fatalf("status after the webhook is registered = %v, %v, want %v", got, err, runtime.ComponentStatusReady)
	}
}
