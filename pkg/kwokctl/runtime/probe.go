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
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
)

const (
	defaultProbeTimeout = time.Second
	maxProbeResponse    = 64 * 1024
)

// ProbeDialFunc opens a connection to the given port of a component.
// It must return when ctx is done. Closing the connection must unblock pending reads and writes.
type ProbeDialFunc func(ctx context.Context, port uint32) (net.Conn, error)

// ComponentStatusWithProbe returns the status of a component based on whether it is running
// and, if a readiness probe is configured, on the result of the probe.
func ComponentStatusWithProbe(ctx context.Context, running bool, probe *internalversion.ComponentProbe, host string, dial ProbeDialFunc) ComponentStatus {
	if !running {
		return ComponentStatusStopped
	}
	if probe == nil {
		return ComponentStatusReady
	}
	err := CheckReadinessProbe(ctx, probe, host, dial)
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Debug("Readiness probe failed",
			"host", host,
			"err", err,
		)
		return ComponentStatusRunning
	}
	return ComponentStatusReady
}

// CheckReadinessProbe performs a single readiness check and returns nil if the component is ready.
// The response is considered ready if its status code is in the range 200-399.
func CheckReadinessProbe(ctx context.Context, probe *internalversion.ComponentProbe, host string, dial ProbeDialFunc) error {
	if probe == nil {
		return errors.New("readiness probe is not configured")
	}
	if probe.HTTPGet == nil {
		return errors.New("readiness probe has no handler")
	}
	if probe.TimeoutSeconds < 0 {
		return fmt.Errorf("invalid readiness probe timeout: %d", probe.TimeoutSeconds)
	}
	port := probe.HTTPGet.Port
	if port == 0 || port > 65535 {
		return fmt.Errorf("invalid readiness probe port: %d", port)
	}
	scheme := probe.HTTPGet.Scheme
	switch scheme {
	case "", internalversion.URISchemeHTTP, internalversion.URISchemeHTTPS:
	default:
		return fmt.Errorf("unsupported readiness probe scheme: %q", scheme)
	}
	request, err := buildProbeRequest(host, port, probe.HTTPGet.Path)
	if err != nil {
		return err
	}

	timeout := defaultProbeTimeout
	if probe.TimeoutSeconds > 0 {
		timeout = time.Duration(probe.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dial(ctx, port)
	if err != nil {
		return probeError(ctx, err)
	}
	defer func() {
		_ = conn.Close()
	}()
	// Closing the connection is what interrupts a relayed transport, so it also bounds every read and write.
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stop()

	var rw io.ReadWriter = conn
	if scheme == internalversion.URISchemeHTTPS {
		tlsConn := tls.Client(conn, probeTLSConfig(host))
		err = tlsConn.HandshakeContext(ctx)
		if err != nil {
			return probeError(ctx, fmt.Errorf("readiness probe TLS handshake: %w", err))
		}
		rw = tlsConn
	}

	_, err = rw.Write(request)
	if err != nil {
		return probeError(ctx, err)
	}
	// The status line and the complete header block are enough to decide, so the body is never read.
	// A truncated header block is rejected.
	resp, err := http.ReadResponse(bufio.NewReader(io.LimitReader(rw, maxProbeResponse)), nil)
	if err != nil {
		return probeError(ctx, fmt.Errorf("invalid readiness probe response: %w", err))
	}
	// Close the connection first, so that closing the body does not wait to drain it.
	_ = conn.Close()
	_ = resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("readiness probe returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// probeError reports a timeout or cancellation of the probe as such,
// since it is what caused the connection to be closed.
func probeError(ctx context.Context, err error) error {
	ctxErr := ctx.Err()
	if ctxErr == nil || errors.Is(err, ctxErr) {
		return err
	}
	return fmt.Errorf("%w: %w", ctxErr, err)
}

// probeTLSConfig returns the TLS configuration of an HTTPS readiness probe.
// Like the kubelet, the probe does not verify the server certificate: components serve certificates that are
// self-signed or signed by the cluster CA, often for names other than the address the probe connects to.
// The probe sends no credentials and only uses the status code, so this exposes nothing to an impersonator.
func probeTLSConfig(host string) *tls.Config {
	config := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
	if net.ParseIP(host) == nil {
		config.ServerName = host
	}
	return config
}

func buildProbeRequest(host string, port uint32, path string) ([]byte, error) {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, " \r\n") {
		return nil, fmt.Errorf("invalid readiness probe path: %q", path)
	}
	if _, err := url.ParseRequestURI(path); err != nil {
		return nil, fmt.Errorf("invalid readiness probe path: %q: %w", path, err)
	}

	// HTTP/1.0 with "Connection: close" makes the server close the connection after the response,
	// so a relayed transport ends with it.
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "GET %s HTTP/1.0\r\n", path)
	_, _ = fmt.Fprintf(&buf, "Host: %s\r\n", net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)))
	buf.WriteString("User-Agent: kwokctl-probe\r\n")
	buf.WriteString("Accept: */*\r\n")
	buf.WriteString("Connection: close\r\n")
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}

// DialProbe returns a ProbeDialFunc that connects directly to host.
func DialProbe(host string) ProbeDialFunc {
	return func(ctx context.Context, port uint32) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)))
	}
}
