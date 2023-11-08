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

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/types"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	remotecommandclient "k8s.io/client-go/tools/remotecommand"
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
	opts := &logOptions{
		tail:      0,
		bytes:     -1, // -1 by default which means read all logs.
		follow:    true,
		timestamp: false,
	}
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
