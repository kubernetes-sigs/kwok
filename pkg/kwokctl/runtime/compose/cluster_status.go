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
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"sigs.k8s.io/kwok/pkg/consts"
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

	host := c.probeHost(status, name)
	return runtime.ComponentStatusWithProbe(ctx, running, component.ReadinessProbe, host, c.probeRoundTrip(host)), nil
}

// probeHost returns the address of the component on the cluster network.
// The container name is a single DNS label, which cannot be resolved once it is longer than 63 characters,
// as it is for long cluster names, so the IP address is preferred.
func (c *Cluster) probeHost(status *inspectStatus, name string) string {
	if status != nil {
		if network, ok := status.NetworkSettings.Networks[c.Name()]; ok && network.IPAddress != "" {
			return network.IPAddress
		}
	}
	return c.Name() + "-" + name
}

// errProbeRelayUnavailable is returned when a readiness probe is configured but cannot be relayed.
var errProbeRelayUnavailable = errors.New("readiness probe requires " + consts.ComponentKwokController + " to be enabled")

// probeRoundTrip returns a function that sends a probe request to host over the cluster network.
// The application image may not contain any network tools, so the request is relayed by nc
// in the kwok-controller container, which shares the cluster network.
func (c *Cluster) probeRoundTrip(host string) runtime.ProbeRoundTripFunc {
	return func(ctx context.Context, port uint32, request []byte) ([]byte, error) {
		// nc -w bounds the connect and the final read inside the container,
		// since killing the exec client does not stop the process in the container.
		timeout := 1
		if deadline, ok := ctx.Deadline(); ok {
			timeout = max(timeout, int(math.Ceil(time.Until(deadline).Seconds())))
		}

		// The input of nc is kept open until the response headers arrive. At the end of its input nc half-closes
		// the connection, and Go servers, such as kubectl proxy, cancel the request when the client half-closes.
		// An *os.File is passed to the process as is, so the input can stay open while the process runs.
		in, inWriter, err := os.Pipe()
		if err != nil {
			return nil, err
		}
		var closeOnce sync.Once
		closeInput := func() {
			closeOnce.Do(func() {
				_ = inWriter.Close()
			})
		}
		// Also end the input when the probe times out, so that the relay never outlives the probe.
		stop := context.AfterFunc(ctx, closeInput)
		defer func() {
			stop()
			closeInput()
			_ = in.Close()
		}()
		// The request is far smaller than the pipe buffer, so this does not block.
		_, err = inWriter.Write(request)
		if err != nil {
			return nil, err
		}

		out := &probeResponseWriter{
			onHeaders: closeInput,
		}
		// Stderr is left unset so that it is included in the returned error.
		ctx = utilsexec.WithIOStreams(ctx, utilsexec.IOStreams{
			In:  in,
			Out: out,
		})
		err = c.Exec(ctx, c.runtime, "exec", "-i", c.Name()+"-"+consts.ComponentKwokController,
			"nc", "-w", strconv.Itoa(timeout), host, strconv.FormatUint(uint64(port), 10))
		return out.Bytes(), err
	}
}

// probeResponseWriter collects a relayed response and calls onHeaders once its header block is complete,
// which is all the probe needs to decide.
// It does not embed the buffer, whose ReadFrom would be used by io.Copy and only return at the end of the output.
type probeResponseWriter struct {
	buf       bytes.Buffer
	onHeaders func()
	done      bool
}

func (w *probeResponseWriter) Write(p []byte) (int, error) {
	n, err := w.buf.Write(p)
	if !w.done && bytes.Contains(w.buf.Bytes(), []byte("\r\n\r\n")) {
		w.done = true
		w.onHeaders()
	}
	return n, err
}

// Bytes returns the response collected so far.
func (w *probeResponseWriter) Bytes() []byte {
	return w.buf.Bytes()
}

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
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string
		}
	}
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
