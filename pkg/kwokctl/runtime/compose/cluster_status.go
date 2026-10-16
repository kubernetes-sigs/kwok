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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	// TODO: Only the necessary components are checked for readiness.
	for _, component := range config.Components {
		s, _ := c.InspectComponent(ctx, component.Name)
		if s != runtime.ComponentStatusReady {
			return false, nil
		}
	}

	return c.Cluster.Ready(ctx)
}

// WaitReady waits for the cluster to be ready.
func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	if c.IsDryRun() {
		return nil
	}
	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.Ready(ctx)
		if err != nil {
			logger.Debug("Cluster is not ready",
				"err", err,
			)
		}
		return ready, nil
	},
		wait.WithTimeout(timeout),
		wait.WithContinueOnError(10),
		wait.WithInterval(time.Second/2),
	)
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// InspectComponent returns the status of the component
func (c *Cluster) InspectComponent(ctx context.Context, name string) (runtime.ComponentStatus, error) {
	status, _ := c.inspect(ctx, name)
	running := status != nil && status.State.Running

	if !running {
		return runtime.ComponentStatusStopped, nil
	}

	component, err := c.GetComponent(ctx, name)
	if err != nil {
		return runtime.ComponentStatusUnknown, err
	}

	if component.ReadinessProbe != nil {
		// Probes are relayed by kwok-controller, so without it the probe cannot succeed.
		// Fail closed, and say why, rather than report a component ready that was never probed.
		_, err = c.GetComponent(ctx, consts.ComponentKwokController)
		if err != nil {
			if errors.Is(err, runtime.ErrComponentNotFound) {
				return runtime.ComponentStatusRunning, fmt.Errorf("%w: %s", errProbeRelayUnavailable, name)
			}
			return runtime.ComponentStatusUnknown, err
		}
	}

	if component.ReadinessProbe == nil {
		return runtime.ComponentStatusReady, nil
	}
	// Arguments that keep the probe from succeeding are reported, rather than probed.
	if err := components.ReadinessProbeConflict(component); err != nil {
		return runtime.ComponentStatusRunning, err
	}
	host, err := c.probeHost(status, name)
	if err != nil {
		return runtime.ComponentStatusRunning, err
	}
	return runtime.ComponentStatusWithProbe(ctx, running, component.ReadinessProbe, host, c.probeDial(host)), nil
}

// maxDNSLabelLength is the longest name that can be resolved as a single DNS label.
const maxDNSLabelLength = 63

// probeHost returns the address of the component on the cluster network.
// The container name is a single DNS label, which cannot be resolved once it is longer than 63 characters,
// as it is for long cluster names, so the IP address is preferred.
func (c *Cluster) probeHost(status *inspectStatus, name string) (string, error) {
	if ip := status.networkAddress(c.Name()); ip != "" {
		return ip, nil
	}
	host := c.Name() + "-" + name
	if len(host) > maxDNSLabelLength {
		return "", fmt.Errorf("%w: %s has no address on network %s, and its container name is too long to resolve",
			errProbeAddressUnavailable, name, c.Name())
	}
	return host, nil
}

// errProbeAddressUnavailable is returned when a readiness probe is configured but its target cannot be addressed.
var errProbeAddressUnavailable = errors.New("readiness probe target has no usable address")

// nerdctlNetworksLabel lists the networks of a nerdctl container, in the order they were attached.
const nerdctlNetworksLabel = "nerdctl/networks"

// networkAddress returns the IP address of the container on the given network, or "" if it has none.
// Docker, Podman and nerdctl since v2.4 key the networks by name. Earlier nerdctl releases key them
// by interface, "unknown-eth<i>" for the i-th network the container was attached to, which is resolved
// through the networks label, as nerdctl itself does since v2.4.
// No other network is ever used, so the address cannot belong to another network.
func (s *inspectStatus) networkAddress(network string) string {
	if s == nil {
		return ""
	}
	endpoint, ok := s.NetworkSettings.Networks[network]
	if !ok {
		var networks []string
		if json.Unmarshal([]byte(s.Config.Labels[nerdctlNetworksLabel]), &networks) == nil {
			if i := slices.Index(networks, network); i >= 0 {
				endpoint, ok = s.NetworkSettings.Networks["unknown-eth"+strconv.Itoa(i)]
			}
		}
	}
	if !ok {
		return ""
	}
	if ip := net.ParseIP(endpoint.IPAddress); ip != nil && ip.To4() != nil && reachableIP(ip) {
		return ip.String()
	}
	if ip := net.ParseIP(endpoint.GlobalIPv6Address); ip != nil && ip.To4() == nil && reachableIP(ip) {
		return ip.String()
	}
	return ""
}

// reachableIP reports whether another container on the network can reach ip.
// A link-local address would also need the zone of the interface, which the inspect result does not have.
func reachableIP(ip net.IP) bool {
	return !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsMulticast()
}

// errProbeRelayUnavailable is returned when a readiness probe is configured but cannot be relayed.
var errProbeRelayUnavailable = errors.New("readiness probe requires " + consts.ComponentKwokController + " to be enabled")

// probeRelayCloseGrace is how long closing a relay waits for nc to exit on its own before the exec is killed.
const probeRelayCloseGrace = time.Second

// probeDial returns a function that connects to host over the cluster network.
// The application image may not contain any network tools, so the connection is relayed by nc
// in the kwok-controller container, which shares the cluster network.
// nc only relays the bytes, so TLS runs over the relay, between kwokctl and the component.
func (c *Cluster) probeDial(host string) runtime.ProbeDialFunc {
	return func(ctx context.Context, port uint32) (net.Conn, error) {
		// nc -w bounds the connect and the final read inside the container,
		// since killing the exec client does not stop the process in the container.
		timeout := 1
		if deadline, ok := ctx.Deadline(); ok {
			timeout = max(timeout, int(math.Ceil(time.Until(deadline).Seconds())))
		}

		// The input of nc is kept open until the connection is closed. At the end of its input nc half-closes
		// the connection, and Go servers, such as kubectl proxy, cancel the request when the client half-closes.
		// An *os.File is passed to the process as is, so the input can stay open while the process runs.
		in, inWriter, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		outReader, out := io.Pipe()

		execCtx, cancel := context.WithCancel(ctx)
		conn := &probeRelayConn{
			in:     inWriter,
			out:    outReader,
			cancel: cancel,
			done:   make(chan struct{}),
		}
		// Stderr is left unset so that it is included in the returned error.
		execCtx = utilsexec.WithIOStreams(execCtx, utilsexec.IOStreams{
			In:  in,
			Out: out,
		})
		go func() {
			defer close(conn.done)
			err := c.Exec(execCtx, c.runtime, "exec", "-i", c.Name()+"-"+consts.ComponentKwokController,
				"nc", "-w", strconv.Itoa(timeout), host, strconv.FormatUint(uint64(port), 10))
			_ = in.Close()
			if err == nil {
				err = io.EOF
			}
			_ = out.CloseWithError(err)
		}()
		return conn, nil
	}
}

// probeRelayConn is a connection relayed through the standard streams of an exec.
type probeRelayConn struct {
	in     *os.File
	out    *io.PipeReader
	cancel context.CancelFunc
	done   chan struct{}

	closeOnce sync.Once
}

func (r *probeRelayConn) Read(p []byte) (int, error) {
	return r.out.Read(p)
}

func (r *probeRelayConn) Write(p []byte) (int, error) {
	return r.in.Write(p)
}

// Close ends the input of nc and waits for the exec to exit,
// killing it if nc has not exited on its own shortly after.
func (r *probeRelayConn) Close() error {
	r.closeOnce.Do(func() {
		_ = r.in.Close()
		// Unblocks the exec if it is writing output that is no longer read.
		_ = r.out.Close()
		timer := time.NewTimer(probeRelayCloseGrace)
		defer timer.Stop()
		select {
		case <-r.done:
		case <-timer.C:
		}
		r.cancel()
		<-r.done
	})
	return nil
}

func (r *probeRelayConn) LocalAddr() net.Addr                { return probeRelayAddr{} }
func (r *probeRelayConn) RemoteAddr() net.Addr               { return probeRelayAddr{} }
func (r *probeRelayConn) SetDeadline(_ time.Time) error      { return errors.ErrUnsupported }
func (r *probeRelayConn) SetReadDeadline(_ time.Time) error  { return errors.ErrUnsupported }
func (r *probeRelayConn) SetWriteDeadline(_ time.Time) error { return errors.ErrUnsupported }

type probeRelayAddr struct{}

func (probeRelayAddr) Network() string { return "nc" }
func (probeRelayAddr) String() string  { return "nc" }

func (c *Cluster) inspectComponent(ctx context.Context, componentName string) (running bool, exist bool) {
	status, exist := c.inspect(ctx, componentName)
	return status != nil && status.State.Running, exist
}

// inspect returns the inspect result of the component container, which is nil if it is unknown.
func (c *Cluster) inspect(ctx context.Context, componentName string) (status *inspectStatus, exist bool) {
	buf := bytes.NewBuffer(nil)
	args := make([]string, 0, 3)
	args = append(args, "inspect", c.Name()+"-"+componentName)

	args = append(args, "--format={{ json . }}")

	err := c.Exec(utilsexec.WithWriteTo(ctx, buf), c.runtime, args...)
	if err != nil {
		// TODO: check if component exists or other error
		return nil, false
	}

	status, err = parseInspect(buf.Bytes())
	if err != nil {
		logger := log.FromContext(ctx)
		logger.Warn("Failed to check inspect result",
			"err", err,
		)
		return nil, false
	}
	return status, true
}

type inspectStatus struct {
	State struct {
		Running bool
	}
	Config struct {
		Labels map[string]string
	}
	NetworkSettings struct {
		Networks map[string]inspectEndpoint
	}
}

type inspectEndpoint struct {
	IPAddress         string
	GlobalIPv6Address string
}

func checkInspect(raw []byte) (bool, error) {
	status, err := parseInspect(raw)
	if err != nil {
		return false, err
	}
	return status != nil && status.State.Running, nil
}

// parseInspect parses an inspect result, which is nil if it is an empty list.
func parseInspect(raw []byte) (*inspectStatus, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty inspect result")
	}
	raw = bytes.TrimSpace(raw)
	switch raw[0] {
	case '{':
		var tmp inspectStatus

		err := json.Unmarshal(raw, &tmp)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal inspect result: %w", err)
		}

		return &tmp, nil
	case '[':
		var tmp []inspectStatus
		err := json.Unmarshal(raw, &tmp)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal inspect result: %w", err)
		}

		if len(tmp) == 0 {
			return nil, nil
		}
		return &tmp[0], nil
	default:
		return nil, fmt.Errorf("unexpected inspect result: %s", raw)
	}
}
