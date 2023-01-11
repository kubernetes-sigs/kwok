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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/types"
	clientremotecommand "k8s.io/client-go/tools/remotecommand"

	"sigs.k8s.io/kwok/pkg/kwok/server/remotecommand"
)

func (s *Server) ExecInContainer(name string, uid types.UID, container string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan clientremotecommand.TerminalSize, timeout time.Duration) error {
	// TODO: Configure and implement the exec streamer
	msg := fmt.Sprintf("TODO: ExecInContainer(%q, %q, %q)", name, container, cmd)
	_, _ = out.Write([]byte(msg))
	return nil
}

func (s *Server) getExec(req *restful.Request, resp *restful.Response) {
	params := getExecRequestParams(req)

	streamOpts, err := remotecommand.NewOptions(req.Request)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	remotecommand.ServeExec(
		resp.ResponseWriter,
		req.Request,
		s,
		params.podNamespace+"/"+params.podName,
		params.podUID,
		params.containerName,
		params.cmd,
		streamOpts,
		s.idleTimeout,
		s.streamCreationTimeout,
		remotecommand.SupportedStreamingProtocols)
}
