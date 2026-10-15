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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func TestPortForwardHalfCloseClientWrite(t *testing.T) {
	stream, target, run := newPortForwardHalfCloseTest(t)
	request := strings.Repeat("request\n", 1024)
	response := strings.Repeat("response\n", 1024)
	run("send request and FIN", func() error {
		if _, err := io.WriteString(stream, request); err != nil {
			return err
		}
		// SPDY Close sends FIN in this direction while preserving reads.
		return stream.Close()
	})
	// The target cannot respond until KWOK propagates the client's FIN.
	got, err := io.ReadAll(target)
	if err != nil {
		t.Fatalf("target waiting for client FIN: %v", err)
	}
	if string(got) != request {
		t.Fatalf("target received %d bytes, want complete %d-byte request", len(got), len(request))
	}
	if _, err := io.WriteString(target, response); err != nil {
		t.Fatal(err)
	}
	if err := target.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	run("receive response after client FIN", func() error {
		got, err = io.ReadAll(stream)
		return err
	})
	if string(got) != response {
		t.Fatalf("client received %d bytes, want complete %d-byte response", len(got), len(response))
	}
}

func TestPortForwardHalfCloseTargetWrite(t *testing.T) {
	stream, target, run := newPortForwardHalfCloseTest(t)
	response := strings.Repeat("response\n", 1024)
	request := strings.Repeat("upload after response EOF\n", 1024)
	if _, err := io.WriteString(target, response); err != nil {
		t.Fatal(err)
	}
	if err := target.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	var got []byte
	run("receive target FIN while client input remains open", func() error {
		var err error
		got, err = io.ReadAll(stream)
		return err
	})
	if string(got) != response {
		t.Fatalf("client received %d bytes, want complete %d-byte response", len(got), len(response))
	}
	// Upload only after receiving EOF, so closing the whole stream instead of
	// its write direction cannot accidentally pass this test.
	run("upload after target FIN", func() error {
		if _, err := io.WriteString(stream, request); err != nil {
			return err
		}
		return stream.Close()
	})
	got, err := io.ReadAll(target)
	if err != nil {
		t.Fatalf("target reading upload after its FIN: %v", err)
	}
	if string(got) != request {
		t.Fatalf("target received %d bytes, want complete %d-byte upload", len(got), len(request))
	}
}

// newPortForwardHalfCloseTest uses the real HTTP route and SPDY stream pair.
// A PortForwarder cannot test target-first FIN: it resets its data stream when
// the remote-to-local copy ends, discarding the remaining upload direction.
func newPortForwardHalfCloseTest(t *testing.T) (httpstream.Stream, *net.TCPConn, func(string, func() error)) {
	t.Helper()
	const timeout = 5 * time.Second
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if err := listener.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
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
	handlerStarted, handlerDone := make(chan struct{}), make(chan struct{})
	var socketLock sync.Mutex
	var socket net.Conn
	stopped := false
	stop := func() {
		socketLock.Lock()
		defer socketLock.Unlock()
		stopped = true
		if socket != nil {
			_ = socket.Close()
		}
	}
	httpServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(handlerStarted)
		defer close(handlerDone)
		s.restfulCont.ServeHTTP(w, r)
	}))
	httpServer.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateHijacked {
			socketLock.Lock()
			defer socketLock.Unlock()
			socket = conn
			if stopped {
				_ = conn.Close()
			}
		}
	}
	httpServer.Start()
	t.Cleanup(func() {
		// httptest leaves hijacked SPDY sockets open. Close the actual socket
		// first so stream reads and writes also unblock on failing paths.
		stop()
		httpServer.Close()
		select {
		case <-handlerStarted:
			select {
			case <-handlerDone:
			case <-time.After(timeout):
				t.Error("port-forward HTTP handler did not stop")
			}
		default:
		}
	})
	run := func(name string, fn func() error) {
		t.Helper()
		done := make(chan error, 1)
		exited := make(chan struct{})
		go func() {
			defer close(exited)
			done <- fn()
		}()
		select {
		case err := <-done:
			<-exited
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		case <-time.After(timeout):
			stop()
			// CreateStream has its own 30-second reply timeout. Even if it
			// has not received an acknowledgement, join its worker before
			// allowing the test's cleanup to return.
			select {
			case <-exited:
			case <-time.After(35 * time.Second):
				t.Errorf("%s worker did not stop after closing the SPDY socket", name)
			}
			t.Fatalf("timed out waiting to %s", name)
		}
	}
	u, err := url.Parse(httpServer.URL + "/portForward/default/test")
	if err != nil {
		t.Fatal(err)
	}
	transport, upgrader, err := spdy.RoundTripperFor(&rest.Config{Host: httpServer.URL})
	if err != nil {
		t.Fatal(err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport, Timeout: timeout}, http.MethodPost, u)
	conn, protocol, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stop()
		_ = conn.Close()
	})
	if protocol != portforward.PortForwardProtocolV1Name {
		t.Fatalf("negotiated protocol %q, want %q", protocol, portforward.PortForwardProtocolV1Name)
	}
	headers := http.Header{}
	headers.Set(v1.StreamType, v1.StreamTypeError)
	headers.Set(v1.PortHeader, "8080")
	headers.Set(v1.PortForwardRequestIDHeader, "0")
	var errorStream, dataStream httpstream.Stream
	run("create error stream", func() error {
		var err error
		errorStream, err = conn.CreateStream(headers)
		return err
	})
	t.Cleanup(func() {
		stop()
		_ = errorStream.Reset()
	})
	run("close error stream input", errorStream.Close)
	headers.Set(v1.StreamType, v1.StreamTypeData)
	run("create data stream", func() error {
		var err error
		dataStream, err = conn.CreateStream(headers)
		return err
	})
	t.Cleanup(func() {
		stop()
		_ = dataStream.Reset()
	})
	// Waiting for the error stream to finish also observes completion of the
	// server's per-stream forwarding handler after both directions reach EOF.
	t.Cleanup(func() {
		if t.Failed() {
			return
		}
		var message []byte
		run("finish forwarding", func() error {
			var err error
			message, err = io.ReadAll(errorStream)
			return err
		})
		if len(message) != 0 {
			t.Errorf("unexpected port-forward error: %s", message)
		}
	})
	target, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = target.Close() })
	if err := target.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	return dataStream, target, run
}
