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

// ProbeRoundTripFunc sends a raw HTTP request to the given port of a component and returns the raw response.
// It must return when ctx is done.
type ProbeRoundTripFunc func(ctx context.Context, port uint32, request []byte) ([]byte, error)

// ComponentStatusWithProbe returns the status of a component based on whether it is running
// and, if a readiness probe is configured, on the result of the probe.
func ComponentStatusWithProbe(ctx context.Context, running bool, probe *internalversion.ComponentProbe, host string, roundTrip ProbeRoundTripFunc) ComponentStatus {
	if !running {
		return ComponentStatusStopped
	}
	if probe == nil {
		return ComponentStatusReady
	}
	err := CheckReadinessProbe(ctx, probe, host, roundTrip)
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
func CheckReadinessProbe(ctx context.Context, probe *internalversion.ComponentProbe, host string, roundTrip ProbeRoundTripFunc) error {
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

	response, rtErr := roundTrip(ctx, port, request)
	// The status line and the complete header block are enough to decide,
	// even if reading the body failed. A truncated header block is rejected.
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(response)), nil)
	if err != nil {
		if rtErr != nil {
			return rtErr
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("invalid readiness probe response: %w", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("readiness probe returned HTTP %d", resp.StatusCode)
	}
	return nil
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
	// so the transport can simply read until EOF.
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "GET %s HTTP/1.0\r\n", path)
	_, _ = fmt.Fprintf(&buf, "Host: %s\r\n", net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)))
	buf.WriteString("User-Agent: kwokctl-probe\r\n")
	buf.WriteString("Accept: */*\r\n")
	buf.WriteString("Connection: close\r\n")
	buf.WriteString("\r\n")
	return buf.Bytes(), nil
}

// DialProbeRoundTrip returns a ProbeRoundTripFunc that connects directly to host.
func DialProbeRoundTrip(host string) ProbeRoundTripFunc {
	return func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10)))
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = conn.Close()
		}()

		stop := context.AfterFunc(ctx, func() {
			_ = conn.SetDeadline(time.Now())
		})
		defer stop()

		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}

		_, err = conn.Write(request)
		if err != nil {
			return nil, err
		}
		return io.ReadAll(io.LimitReader(conn, maxProbeResponse))
	}
}
