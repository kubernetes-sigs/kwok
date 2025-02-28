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
	crilogs "k8s.io/cri-client/pkg/logs"
	remotecommandserver "k8s.io/kubelet/pkg/cri/streaming/remotecommand"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// AttachContainer attaches to a container in a pod,
// copying data between in/out/err and the container's stdin/stdout/stderr.
func (s *Server) AttachContainer(ctx context.Context, name string, uid types.UID, containerName string, in io.Reader, out, errOut io.WriteCloser, tty bool, resize <-chan remotecommandclient.TerminalSize) error {
	pod := strings.Split(name, "/")
	if len(pod) != 2 {
		return fmt.Errorf("invalid pod name %q", name)
	}
	podName, podNamespace := pod[0], pod[1]
	attach, err := getPodAttach(s.attaches.Get(), s.clusterAttaches.Get(), podName, podNamespace, containerName)
	if err != nil {
		return err
	}

	if m := attach.Mapping; m != nil {
		return s.attachMappingToContainer(ctx, m.Namespace, m.Name, m.Container, in, out, errOut, tty, resize)
	}

	var tailLines int64
	opts := crilogs.NewLogOptions(&corev1.PodLogOptions{
		TailLines: &tailLines,
		Follow:    true,
	}, time.Now())
	return readLogs(ctx, attach.LogsFile, opts, out, errOut)
}

func (s *Server) getAttach(req *restful.Request, resp *restful.Response) {
	params := getExecRequestParams(req)

	streamOpts, err := remotecommandserver.NewOptions(req.Request)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	remotecommandserver.ServeAttach(
		resp.ResponseWriter,
		req.Request,
		s,
		params.podName+"/"+params.podNamespace,
		params.podUID,
		params.containerName,
		streamOpts,
		s.idleTimeout,
		s.streamCreationTimeout,
		remotecommandconsts.SupportedStreamingProtocols,
	)
}

func getPodAttach(rules []*internalversion.Attach, clusterRules []*internalversion.ClusterAttach, podName, podNamespace, containerName string) (*internalversion.AttachConfig, error) {
	a, has := slices.Find(rules, func(a *internalversion.Attach) bool {
		return a.Name == podName && a.Namespace == podNamespace
	})
	if has {
		a, found := findAttachInAttaches(containerName, a.Spec.Attaches)
		if found {
			return a, nil
		}
		return nil, fmt.Errorf("attaches target not found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
	}

	for _, cl := range clusterRules {
		if !cl.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		a, found := findAttachInAttaches(containerName, cl.Spec.Attaches)
		if found {
			return a, nil
		}
	}

	return nil, fmt.Errorf("no attaches found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
}

func findAttachInAttaches(containerName string, attaches []internalversion.AttachConfig) (*internalversion.AttachConfig, bool) {
	var defaultAttach *internalversion.AttachConfig
	for i, a := range attaches {
		if len(a.Containers) == 0 && defaultAttach == nil {
			defaultAttach = &attaches[i]
			continue
		}
		if slices.Contains(a.Containers, containerName) {
			return &a, true
		}
	}
	return defaultAttach, defaultAttach != nil
}

// attachMappingToContainer attaches to a container in a pod,
// copying data between in/out/err and the container's stdin/stdout/stderr.
func (s *Server) attachMappingToContainer(ctx context.Context, namespace, name, container string, in io.Reader, out, errOut io.WriteCloser, tty bool, resize <-chan remotecommandclient.TerminalSize) error {
	req := s.typedClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("attach")

	attachOptions := &corev1.PodAttachOptions{
		Container: container,
		TTY:       tty,
		Stdin:     in != nil,
		Stdout:    out != nil,
		Stderr:    errOut != nil,
	}

	req.VersionedParams(attachOptions, scheme.ParameterCodec)

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
		return fmt.Errorf("unable to stream: %w", err)
	}
	return nil
}

func newTranslatorSizeQueue(resize <-chan remotecommandclient.TerminalSize) remotecommandclient.TerminalSizeQueue {
	return &translatorSizeQueue{
		resizeChan: resize,
	}
}

// translatorSizeQueue feeds the size events from the WebSocket
// resizeChan into the SPDY client input. Implements TerminalSizeQueue
// interface.
type translatorSizeQueue struct {
	resizeChan <-chan remotecommandclient.TerminalSize
}

func (t *translatorSizeQueue) Next() *remotecommandclient.TerminalSize {
	size, ok := <-t.resizeChan
	if !ok {
		return nil
	}
	return &size
}
