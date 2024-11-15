/*
Copyright 2023 The Kubernetes Authors.

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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	"k8s.io/client-go/kubernetes/scheme"
	remotecommandclient "k8s.io/client-go/tools/remotecommand"
	remotecommandserver "k8s.io/kubelet/pkg/cri/streaming/remotecommand"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// ExecInContainer executes a command in a container.
func (s *Server) ExecInContainer(ctx context.Context, name string, uid types.UID, container string, cmd []string, in io.Reader, out, errOut io.WriteCloser, tty bool, resize <-chan remotecommandclient.TerminalSize, timeout time.Duration) error {
	pod := strings.Split(name, "/")
	if len(pod) != 2 {
		return fmt.Errorf("invalid pod name %q", name)
	}
	podName, podNamespace := pod[0], pod[1]
	execTarget, err := getExecTarget(s.execs.Get(), s.clusterExecs.Get(), podName, podNamespace, container)
	if err != nil {
		return err
	}

	if m := execTarget.Mapping; m != nil {
		return s.execMappingToContainer(ctx, m.Namespace, m.Name, m.Container, cmd, in, out, errOut, tty, resize, timeout)
	}

	// Currently only support local exec.
	if execTarget.Local == nil {
		return fmt.Errorf("not set local exec")
	}

	// Set the environment variables.
	if len(execTarget.Local.Envs) != 0 {
		envs := slices.Map(execTarget.Local.Envs, func(env internalversion.EnvVar) string {
			return fmt.Sprintf("%s=%s", env.Name, env.Value)
		})
		ctx = exec.WithEnv(ctx, envs)
	}

	// Set the user.
	if execTarget.Local.SecurityContext != nil {
		ctx = exec.WithUser(ctx, execTarget.Local.SecurityContext.RunAsUser, execTarget.Local.SecurityContext.RunAsGroup)
	}

	// Set the working directory.
	if execTarget.Local.WorkDir != "" {
		ctx = exec.WithDir(ctx, execTarget.Local.WorkDir)
	}

	// Set cancel context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if tty {
		return s.execInContainerWithTTY(ctx, cmd, in, out, resize)
	}

	return s.execInContainer(ctx, cmd, in, out, errOut)
}

func (s *Server) execInContainer(ctx context.Context, cmd []string, in io.Reader, out, errOut io.WriteCloser) error {
	// Set the pipe stdin.
	if in != nil {
		ctx = exec.WithPipeStdin(ctx, true)
	}

	// Set the stream as the stdin/stdout/stderr.
	ctx = exec.WithIOStreams(ctx, exec.IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	})

	// Execute the command.
	err := exec.Exec(ctx, cmd[0], cmd[1:]...)
	if err != nil {
		return err
	}
	return nil
}

func getExecTarget(rules []*internalversion.Exec, clusterRules []*internalversion.ClusterExec, podName, podNamespace string, containerName string) (*internalversion.ExecTarget, error) {
	e, has := slices.Find(rules, func(pf *internalversion.Exec) bool {
		return pf.Name == podName && pf.Namespace == podNamespace
	})
	if has {
		exec, found := findContainerInExecs(containerName, e.Spec.Execs)
		if found {
			return exec, nil
		}
		return nil, fmt.Errorf("exec target not found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
	}

	for _, ce := range clusterRules {
		if !ce.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		exec, found := findContainerInExecs(containerName, ce.Spec.Execs)
		if found {
			return exec, nil
		}
	}
	return nil, fmt.Errorf("no exec found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
}

func findContainerInExecs(containerName string, execs []internalversion.ExecTarget) (*internalversion.ExecTarget, bool) {
	var defaultExecTarget *internalversion.ExecTarget
	for i, ex := range execs {
		if len(ex.Containers) == 0 && defaultExecTarget == nil {
			defaultExecTarget = &execs[i]
			continue
		}
		if slices.Contains(ex.Containers, containerName) {
			return &ex, true
		}
	}
	return defaultExecTarget, defaultExecTarget != nil
}

func (s *Server) getExec(req *restful.Request, resp *restful.Response) {
	params := getExecRequestParams(req)

	streamOpts, err := remotecommandserver.NewOptions(req.Request)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	remotecommandserver.ServeExec(
		resp.ResponseWriter,
		req.Request,
		s,
		params.podName+"/"+params.podNamespace,
		params.podUID,
		params.containerName,
		params.cmd,
		streamOpts,
		s.idleTimeout,
		s.streamCreationTimeout,
		remotecommandconsts.SupportedStreamingProtocols,
	)
}

// execMappingToContainer executes a command in a container with mapping.
func (s *Server) execMappingToContainer(ctx context.Context, namespace, name, container string, cmd []string, in io.Reader, out, errOut io.WriteCloser, tty bool, resize <-chan remotecommandclient.TerminalSize, timeout time.Duration) error {
	req := s.typedClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("exec").
		Timeout(timeout)

	execOptions := &corev1.PodExecOptions{
		Command:   cmd,
		Container: container,
		TTY:       tty,
		Stdin:     in != nil,
		Stdout:    out != nil,
		Stderr:    errOut != nil,
	}

	req.VersionedParams(execOptions, scheme.ParameterCodec)

	executor, err := remotecommandclient.NewSPDYExecutor(s.restConfig, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("unable to create executor: %w", err)
	}

	err = executor.StreamWithContext(ctx, remotecommandclient.StreamOptions{
		Stdin:             in,
		Stdout:            out,
		Stderr:            errOut,
		Tty:               tty,
		TerminalSizeQueue: newTranslatorSizeQueue(resize),
	})
	if err != nil {
		return fmt.Errorf("unable to stream output: %w", err)
	}
	return nil
}
