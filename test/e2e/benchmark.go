/*
Copyright 2024 The Kubernetes Authors.

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

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// kwokctlStopGrace bounds how long a stopped kwokctl may take to exit on its own
// before it is killed and the harness-side ends of its pipes are closed.
const kwokctlStopGrace = 5 * time.Second

// kwokctlProcess is a kwokctl process started by the benchmark harness.
//
// Its context is derived from the owner's, so the owner's end interrupts it, and
// stop does the same deliberately; both paths end with the process reaped and the
// harness-side pipe ends closed within kwokctlStopGrace. kwokctl handles
// os.Interrupt by killing its own kubectl/kectl child, which exec.Cmd's default
// Kill would orphan with the stdout pipe still open.
//
// On windows Close does not unblock a pending Read on the pipe; the benchmark suites only run on linux.
type kwokctlProcess struct {
	name   string
	ctx    context.Context
	runCtx context.Context
	cancel context.CancelFunc
	cmd    *exec.Cmd

	// interrupted is set by cmd.Cancel once its signal reached the live process.
	interrupted bool

	done chan struct{}
	err  error

	childStdout *os.File
	pipes       []io.Closer
	closeOnce   sync.Once
}

func newKwokctl(ctx context.Context, kwokctlPath string, args ...string) *kwokctlProcess {
	runCtx, cancel := context.WithCancel(ctx)
	p := &kwokctlProcess{
		name:   "kwokctl " + strings.Join(args, " "),
		ctx:    ctx,
		runCtx: runCtx,
		cancel: cancel,
		cmd:    exec.CommandContext(runCtx, kwokctlPath, args...),
		done:   make(chan struct{}),
	}
	p.cmd.Cancel = func() error {
		err := interruptOrKill(p.cmd.Process)
		if err == nil {
			p.interrupted = true
		}
		return err
	}
	p.cmd.WaitDelay = kwokctlStopGrace
	return p
}

// interruptOrKill sends the one graceful signal; a second one would make kwokctl
// exit before killing its child. Where os.Interrupt cannot be delivered
// (windows) the process is killed instead; one already done is left alone.
func interruptOrKill(p *os.Process) error {
	err := p.Signal(os.Interrupt)
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return p.Kill()
}

// stdoutPipe connects the process's stdout to a pipe and returns the read end.
// Unlike exec.Cmd.StdoutPipe, the read end survives the reap until the process
// is stopped, so a natural EOF is drained in full, and stop closes it even while
// an orphaned grandchild holds the write end, which frees a blocked reader.
func (p *kwokctlProcess) stdoutPipe() (io.Reader, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	p.cmd.Stdout = pw
	p.childStdout = pw
	p.pipes = append(p.pipes, pr)
	return pr, nil
}

// start runs the process and reaps it exactly once in the background.
func (p *kwokctlProcess) start() error {
	err := p.cmd.Start()
	if p.childStdout != nil {
		_ = p.childStdout.Close()
	}
	if err != nil {
		p.err = err
		close(p.done)
		p.cancel()
		p.closePipes()
		return err
	}
	go func() {
		err := p.cmd.Wait()
		if p.interrupted && p.ctx.Err() != nil {
			// kwokctl exits 1 after the interrupt the owner's end sent; report the cause.
			err = p.ctx.Err()
		}
		p.err = err
		close(p.done)
		<-p.runCtx.Done()
		p.closePipes()
	}()
	return nil
}

func (p *kwokctlProcess) closePipes() {
	p.closeOnce.Do(func() {
		for _, c := range p.pipes {
			_ = c.Close()
		}
	})
}

// wait blocks until the process is reaped and returns its classified exit.
func (p *kwokctlProcess) wait() error {
	<-p.done
	return p.err
}

// failure names a process that exited unsuccessfully on its own; it is nil while
// the process runs, after a clean or context-forced exit, and for a nil process.
func (p *kwokctlProcess) failure() error {
	if p == nil {
		return nil
	}
	select {
	case <-p.done:
	default:
		return nil
	}
	if p.err == nil || errors.Is(p.err, context.Canceled) || errors.Is(p.err, context.DeadlineExceeded) {
		return nil
	}
	return fmt.Errorf("%s: %w", p.name, p.err)
}

// stop interrupts the process, waits at most kwokctlStopGrace for its exit and
// closes the harness-side pipe ends. It is idempotent and a no-op for nil.
func (p *kwokctlProcess) stop() {
	stopAll(p)
}

// stopAll stops several processes under one shared grace.
func stopAll(ps ...*kwokctlProcess) {
	for _, p := range ps {
		if p != nil {
			p.cancel()
		}
	}
	grace, cancel := context.WithTimeout(context.Background(), kwokctlStopGrace)
	defer cancel()
	for _, p := range ps {
		if p == nil {
			continue
		}
		select {
		case <-p.done:
		case <-grace.Done():
		}
		p.closePipes()
	}
}

// kwokctlOutput runs kwokctl to completion and returns its stdout.
func kwokctlOutput(ctx context.Context, kwokctlPath string, args ...string) ([]byte, error) {
	var out bytes.Buffer
	p := newKwokctl(ctx, kwokctlPath, args...)
	p.cmd.Stdout = &out
	p.cmd.Stderr = os.Stderr
	if err := p.start(); err != nil {
		return nil, err
	}
	defer p.stop()
	if err := p.wait(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// startScale starts a kwokctl scale command with its output on stderr.
func startScale(ctx context.Context, kwokctlPath string, args ...string) (*kwokctlProcess, error) {
	p := newKwokctl(ctx, kwokctlPath, args...)
	p.cmd.Stdout = os.Stderr
	p.cmd.Stderr = os.Stderr
	if err := p.start(); err != nil {
		return nil, err
	}
	return p, nil
}

// sleepCtx pauses for d or until ctx ends, returning the context's error then.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func waitDeleteResource(ctx context.Context, t *testing.T, kwokctlPath, name, resource string, driver *kwokctlProcess) error {
	for {
		data, err := kwokctlOutput(ctx, kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", resource)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			return nil
		}
		if err := driver.failure(); err != nil {
			return err
		}

		got := bytes.Count(data, []byte{'\n'})

		t.Logf("%s %d\n", resource, got)
		if err := sleepCtx(ctx, time.Second); err != nil {
			return err
		}
	}
}

func waitResource(ctx context.Context, t *testing.T, kwokctlPath, name, resource, reason string, want, gap, tolerance int, startFunc func() (*kwokctlProcess, error)) error {
	startWatch := func() (*kwokctlProcess, *bufio.Reader, error) {
		watch := newKwokctl(ctx, kwokctlPath, "--name", name, "kubectl", "get", "--no-headers", "--watch", resource)
		watch.cmd.Stderr = os.Stderr
		pr, err := watch.stdoutPipe()
		if err != nil {
			return nil, nil, err
		}
		if err := watch.start(); err != nil {
			return nil, nil, err
		}
		return watch, bufio.NewReader(pr), nil
	}

	watch, reader, err := startWatch()
	if err != nil {
		return err
	}
	var driver *kwokctlProcess
	defer func() {
		stopAll(watch, driver)
	}()

	uniq := map[string]int{}
	prev := 0
	got := 0
	var latestTime time.Time

	ctxEnded := func() error {
		stopAll(watch, driver)
		all := len(uniq)
		t.Logf("%s %d/%d => %d, %v\n", resource, got, all, want, time.Now())
		err := fmt.Errorf("%w: %s %d/%d => %d", ctx.Err(), resource, got, all, want)
		if failure := driver.failure(); failure != nil {
			err = fmt.Errorf("%w (%v)", err, failure)
		}
		return err
	}

	if startFunc != nil {
		driver, err = startFunc()
		if err != nil {
			return err
		}
		if sleepCtx(ctx, time.Second) != nil {
			return ctxEnded()
		}
	}

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			// The stream ended: EOF, or the read end closed once a stopped watch was reaped.
			if ctx.Err() != nil {
				return ctxEnded()
			}
			t.Logf("EOF: %s %d => %d, %v\n", resource, got, want, latestTime)

			watch.stop()

			if sleepCtx(ctx, time.Second) != nil {
				return ctxEnded()
			}

			watch, reader, err = startWatch()
			if err != nil {
				if ctx.Err() != nil {
					return ctxEnded()
				}
				return err
			}
			continue
		}

		key := string(line[:bytes.IndexByte(line, byte(' '))])

		_, ok := uniq[key]
		if !ok {
			uniq[key] = 0
		}

		if uniq[key] == 0 {
			if bytes.Contains(line, []byte(reason)) {
				uniq[key] = 1
				got++
			}
		}

		if got == want {
			t.Logf("%s %d, %v\n", resource, got, latestTime)
			return nil
		}

		if time.Since(latestTime) < time.Second {
			continue
		}

		latestTime = time.Now()

		all := len(uniq)

		t.Logf("%s %d/%d => %d, %v\n", resource, got, all, want, latestTime)
		if err := driver.failure(); err != nil {
			return err
		}
		if prev != 0 && got == prev {
			return fmt.Errorf("resource %s not changed", resource)
		}
		prev = got
		if gap != 0 && got != 0 && (all-got) > gap {
			if tolerance > 0 {
				t.Logf("Error %s gap too large, actual: %d, expected: %d, retrying...\n", resource, all-got, gap)
				tolerance--
			} else {
				t.Logf("Error %s gap too large, actual: %d, expected: %d\n", resource, all-got, gap)
				return fmt.Errorf("gap too large for resource %s", resource)
			}
		}
	}
}

func scaleCreatePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, size, gap, tolerance int) error {
	nodeName := "fake-node-000000"
	startScaleCmd := func() (*kwokctlProcess, error) {
		return startScale(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(size), "--param", fmt.Sprintf(".nodeName=%q", nodeName))
	}

	if err := waitResource(ctx, t, kwokctlPath, name, "Pod", "Running", size, gap, tolerance, startScaleCmd); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleDeletePod(ctx context.Context, t *testing.T, kwokctlPath string, name string, _ int) error {
	scaleCmd, err := startScale(ctx, kwokctlPath, "--name", name, "scale", "pod", "fake-pod", "--replicas", strconv.Itoa(0))
	if err != nil {
		return err
	}
	defer scaleCmd.stop()

	if err := waitDeleteResource(ctx, t, kwokctlPath, name, "Pod", scaleCmd); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

func scaleCreateNode(ctx context.Context, t *testing.T, kwokctlPath string, name string, size, gap, tolerance int) error {
	startScaleCmd := func() (*kwokctlProcess, error) {
		return startScale(ctx, kwokctlPath, "--name", name, "scale", "node", "fake-node", "--replicas", strconv.Itoa(size))
	}

	if err := waitResource(ctx, t, kwokctlPath, name, "Node", "Ready", size, gap, tolerance, startScaleCmd); err != nil {
		return fmt.Errorf("failed to wait for resource: %w", err)
	}
	return nil
}

// CaseBenchmark defines a feature test suite for benchmarking the performance of a KWOK cluster.
func CaseBenchmark(kwokctlPath, clusterName string) *features.FeatureBuilder {
	return features.New("Benchmark Hack").
		Assess("Create nodes", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			err := scaleCreateNode(ctx0, t, kwokctlPath, clusterName, 5000, 5, 5)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Create pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			err := scaleCreatePod(ctx0, t, kwokctlPath, clusterName, 10000, 5, 5)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Delete pods", func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
			ctx0, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()

			err := scaleDeletePod(ctx0, t, kwokctlPath, clusterName, 10000)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
