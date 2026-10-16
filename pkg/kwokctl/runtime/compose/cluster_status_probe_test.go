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
	"os"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
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
	for _, s := range []string{"running", "stopped", "ready", "ip"} {
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
	// Without an address on the cluster network, the container name is used, and the probe still fails closed.
	f := newFakeComposeWithName(t, "kwokctl-readiness-docker-1-37-0", testKwokController, testKubectlProxy)
	f.set(t, consts.ComponentKwokController, "running")
	f.set(t, consts.ComponentKubeApiserverInsecureProxy, "running", "ready")

	got, err := f.inspect(t, consts.ComponentKubeApiserverInsecureProxy)
	if err != nil || got != runtime.ComponentStatusRunning {
		t.Fatalf("status = %v, %v, want %v", got, err, runtime.ComponentStatusRunning)
	}
	want := []string{f.Name() + "-kwok-controller nc " + f.Name() + "-kube-apiserver-insecure-proxy:8001"}
	if got := f.requests(t); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("relayed requests = %q, want %q", got, want)
	}
}
