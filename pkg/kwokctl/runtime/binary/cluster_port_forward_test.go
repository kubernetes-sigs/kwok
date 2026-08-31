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
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
)

// pickUnusedPort asks the OS for an ephemeral TCP port and releases it
// immediately so the caller can reuse the port number.
func pickUnusedPort(t *testing.T) uint32 {
	t.Helper()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to allocate an unused port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release allocated port: %v", err)
	}
	return uint32(port)
}

// TestPortForwardRecoversAfterFailedTargetDial is a regression test for
// https://github.com/kubernetes-sigs/kwok/issues/1740: a single failed dial
// to the target port used to terminate the whole accept loop, permanently
// killing the port-forward while the listening socket stayed open.
func TestPortForwardRecoversAfterFailedTargetDial(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelCtx()

	hostPort := pickUnusedPort(t)
	targetPort := pickUnusedPort(t)
	for targetPort == hostPort {
		targetPort = pickUnusedPort(t)
	}
	// nothing listens on targetPort yet

	c := &Cluster{
		Cluster: runtime.NewCluster("test", t.TempDir()),
	}

	cancel, err := c.PortForward(ctx, "test-component", strconv.FormatUint(uint64(targetPort), 10), hostPort)
	if err != nil {
		t.Fatalf("PortForward returned error: %v", err)
	}
	defer cancel()

	fwdAddr := fmt.Sprintf("127.0.0.1:%d", hostPort)

	// Step 1: connect once while nothing listens on targetPort, forcing the
	// internal dial to the target to fail.
	conn1, err := net.DialTimeout("tcp", fwdAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to dial forwarding port: %v", err)
	}
	defer func() { _ = conn1.Close() }()

	// Step 2: deterministically prove the failed dial has already been
	// processed. Both the buggy and fixed implementations close this first
	// connection immediately after the target dial fails (only what happens
	// to the accept loop afterward differs), so observing the close here -
	// bounded by a deadline instead of an arbitrary sleep - is a reliable,
	// version-independent synchronization point.
	if err := conn1.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := conn1.Read(buf); err == nil {
		t.Fatalf("expected connection to be closed after failed target dial, but read succeeded")
	}
	_ = conn1.Close()

	// Step 3: bring the target up and prove the SAME PortForward instance
	// can still serve a new connection.
	targetLn, err := net.Listen("tcp", fmt.Sprintf(":%d", targetPort))
	if err != nil {
		t.Fatalf("failed to start target listener: %v", err)
	}
	defer func() { _ = targetLn.Close() }()

	go func() {
		conn, err := targetLn.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = io.Copy(conn, conn) // echo
	}()

	conn2, err := net.DialTimeout("tcp", fwdAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("failed to dial forwarding port a second time: %v", err)
	}
	defer func() { _ = conn2.Close() }()

	if err := conn2.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set deadline: %v", err)
	}

	// merely establishing conn2 is not sufficient proof of recovery: on the
	// buggy implementation the kernel listen backlog can still complete a
	// TCP handshake even though nothing calls Accept anymore. Actual data
	// must flow end-to-end through the forward.
	const payload = "ping"
	if _, err := conn2.Write([]byte(payload)); err != nil {
		t.Fatalf("failed to write to forwarded connection: %v", err)
	}

	resp := make([]byte, len(payload))
	if _, err := io.ReadFull(conn2, resp); err != nil {
		t.Fatalf("failed to read echoed data through recovered PortForward: %v", err)
	}
	if string(resp) != payload {
		t.Fatalf("unexpected echoed data: got %q, want %q", resp, payload)
	}
}
