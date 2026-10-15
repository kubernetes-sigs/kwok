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

package net

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

const tcpTunnelTestTimeout = 5 * time.Second

func tunnelTCPPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	if err := listener.SetDeadline(time.Now().Add(tcpTunnelTestTimeout)); err != nil {
		t.Fatal(err)
	}
	dialer := net.Dialer{Timeout: tcpTunnelTestTimeout}
	conn, err := dialer.DialContext(t.Context(), "tcp4", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	peer, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = peer.Close() })
	return conn.(*net.TCPConn), peer
}

// newTCPHalfCloseTunnel uses two real TCP connections and closes the proxy
// sockets after Tunnel returns, as its network callers do.
func newTCPHalfCloseTunnel(t *testing.T) (left, right, proxyLeft, proxyRight *net.TCPConn, result <-chan error) {
	t.Helper()
	left, proxyLeft = tunnelTCPPair(t)
	proxyRight, right = tunnelTCPPair(t)
	conns := []*net.TCPConn{left, right, proxyLeft, proxyRight}
	for _, conn := range conns {
		if err := conn.SetDeadline(time.Now().Add(tcpTunnelTestTimeout)); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		err := Tunnel(ctx, proxyLeft, proxyRight, nil)
		_ = proxyLeft.Close()
		_ = proxyRight.Close()
		done <- err
	}()
	t.Cleanup(func() {
		cancel()
		for _, conn := range conns {
			_ = conn.Close()
		}
		select {
		case <-exited:
		case <-time.After(tcpTunnelTestTimeout):
			t.Error("Tunnel did not exit after its sockets were closed")
		}
	})
	return left, right, proxyLeft, proxyRight, done
}

func TestTunnelHalfClosePropagatesEOF(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "first stream"
		if reverse {
			name = "second stream"
		}
		t.Run(name, func(t *testing.T) {
			testTCPHalfCloseExchange(t, reverse, false)
		})
	}
}

func TestTunnelHalfCloseAfterReadTimeout(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "first stream"
		if reverse {
			name = "second stream"
		}
		t.Run(name, func(t *testing.T) {
			testTCPHalfCloseExchange(t, reverse, true)
		})
	}
}

func testTCPHalfCloseExchange(t *testing.T, reverse, readTimeout bool) {
	t.Helper()
	left, right, proxyLeft, proxyRight, done := newTCPHalfCloseTunnel(t)
	if reverse {
		left, right = right, left
		proxyLeft = proxyRight
	}
	request := []byte("request delimited by EOF")
	if _, err := left.Write(request); err != nil {
		t.Fatal(err)
	}
	gotRequest := make([]byte, len(request))
	if _, err := io.ReadFull(right, gotRequest); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotRequest, request) {
		t.Fatalf("request = %q, want %q", gotRequest, request)
	}
	if readTimeout {
		// Expire only this read direction after the request has arrived. A read
		// deadline does not prevent this socket from writing the response.
		if err := proxyLeft.SetReadDeadline(time.Now()); err != nil {
			t.Fatal(err)
		}
	} else if err := left.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	// The peer cannot generate its response until the request's EOF arrives.
	if remaining, err := io.ReadAll(right); err != nil {
		t.Fatalf("request EOF was not propagated: %v", err)
	} else if len(remaining) != 0 {
		t.Fatalf("unexpected request suffix: %q", remaining)
	}
	response := bytes.Repeat([]byte("response payload\n"), 8192)
	writeDone := make(chan error, 1)
	writeExited := make(chan struct{})
	go func() {
		defer close(writeExited)
		if _, err := right.Write(response); err != nil {
			writeDone <- err
			return
		}
		writeDone <- right.CloseWrite()
	}()
	t.Cleanup(func() {
		_ = right.Close()
		select {
		case <-writeExited:
		case <-time.After(tcpTunnelTestTimeout):
			t.Error("response writer did not exit after its socket was closed")
		}
	})
	gotResponse, err := io.ReadAll(left)
	if err != nil {
		t.Fatalf("read response after half-close: %v", err)
	}
	if !bytes.Equal(gotResponse, response) {
		t.Fatalf("response after half-close has %d bytes, want %d", len(gotResponse), len(response))
	}
	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write response after half-close: %v", err)
		}
	case <-time.After(tcpTunnelTestTimeout):
		t.Fatal("response writer did not finish")
	}
	select {
	case err := <-done:
		if readTimeout {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("Tunnel returned %v, want the read deadline error", err)
			}
		} else if err != nil {
			t.Fatalf("Tunnel returned %v after a normal half-close", err)
		}
	case <-time.After(tcpTunnelTestTimeout):
		t.Fatal("Tunnel did not return after both directions finished")
	}
}
