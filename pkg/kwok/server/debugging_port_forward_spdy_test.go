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

package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestPortForwardClosesClientAfterTargetReset(t *testing.T) {
	const timeout = 5 * time.Second
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	s, err := NewServer(Config{PortForwards: []*internalversion.PortForward{{
		Name: "test", Namespace: "default",
		Spec: internalversion.PortForwardSpec{Forwards: []internalversion.Forward{{
			Ports:  []int32{8080},
			Target: &internalversion.ForwardTarget{Address: "127.0.0.1", Port: int32(listener.Addr().(*net.TCPAddr).Port)},
		}}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	s.streamCreationTimeout = timeout
	s.InstallDebuggingHandlers()
	handlerDone := make(chan struct{})
	hijacked := make(chan net.Conn, 1)
	httpServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		s.restfulCont.ServeHTTP(w, r)
	}))
	httpServer.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateHijacked {
			hijacked <- conn
		}
	}
	httpServer.Start()
	t.Cleanup(func() {
		// httptest does not close connections hijacked by the SPDY upgrader.
		select {
		case conn := <-hijacked:
			_ = conn.Close()
			select {
			case <-handlerDone:
			case <-time.After(timeout):
				t.Error("port-forward HTTP handler did not stop")
			}
		default:
		}
		httpServer.Close()
	})

	u, err := url.Parse(httpServer.URL + "/portForward/default/test")
	if err != nil {
		t.Fatal(err)
	}
	transport, upgrader, err := spdy.RoundTripperFor(&rest.Config{Host: httpServer.URL})
	if err != nil {
		t.Fatal(err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport, Timeout: 2 * timeout}, http.MethodPost, u)
	stop, ready := make(chan struct{}), make(chan struct{})
	client, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{"0:8080"}, stop, ready, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	forwardDone := make(chan error, 1)
	forwardExited := make(chan struct{})
	go func() {
		defer close(forwardExited)
		forwardDone <- client.ForwardPorts()
	}()
	t.Cleanup(func() {
		close(stop)
		select {
		case <-forwardExited:
		case <-time.After(2 * timeout):
			t.Error("client port-forwarder did not stop")
		}
	})
	select {
	case <-ready:
	case err := <-forwardDone:
		t.Fatalf("client port-forwarder stopped before becoming ready: %v", err)
	case <-time.After(timeout):
		t.Fatal("client port-forwarder did not become ready")
	}
	ports, err := client.GetPorts()
	if err != nil {
		t.Fatal(err)
	}
	local, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", ports[0].Local), timeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = local.Close() })
	if err := local.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	const request = "request"
	if _, err := io.WriteString(local, request); err != nil {
		t.Fatal(err)
	}
	if err := listener.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	target, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = target.Close() })
	if err := target.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(request))
	if _, err := io.ReadFull(target, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != request {
		t.Fatalf("target received %q, want %q", buf, request)
	}
	// Reset the actual target socket while the client's sending direction stays
	// open. Waiting for that direction before reporting the reset hangs the client.
	if err := target.SetLinger(0); err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	if err := local.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	if n, err := local.Read(buf[:1]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("client Read after target reset = (%d, %v), want (0, EOF) without closing client input", n, err)
	}
	// The real SPDY error stream should propagate the target failure and stop
	// the client-go session without the test closing its stop channel.
	select {
	case err := <-forwardDone:
		if !errors.Is(err, portforward.ErrLostConnectionToPod) {
			t.Fatalf("ForwardPorts returned %v, want %v", err, portforward.ErrLostConnectionToPod)
		}
	case <-time.After(timeout):
		t.Fatal("client port-forwarder did not report the target failure")
	}
}
