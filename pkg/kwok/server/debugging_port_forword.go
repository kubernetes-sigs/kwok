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

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/kwok/pkg/kwok/server/portforward"
	"sigs.k8s.io/kwok/pkg/log"
)

func (s *Server) PortForward(name string, uid types.UID, port int32, stream io.ReadWriteCloser) error {
	// TODO: Configure and implement the port forward streamer
	msg := fmt.Sprintf("TODO: PortForward(%q, %q)", name, port)
	_, _ = stream.Write([]byte(msg))
	return nil
}

// getPortForward handles a new restful port forward request. It determines the
// pod name and uid and then calls ServePortForward.
func (s *Server) getPortForward(req *restful.Request, resp *restful.Response) {
	params := getPortForwardRequestParams(req)

	portForwardOptions, err := portforward.NewV4Options(req.Request)
	if err != nil {
		logger := log.FromContext(req.Request.Context())
		logger.Error("NewV4Options", err)
		_ = resp.WriteError(http.StatusBadRequest, err)
		return
	}

	portforward.ServePortForward(
		resp.ResponseWriter,
		req.Request,
		s,
		params.podNamespace+"/"+params.podName,
		params.podUID,
		portForwardOptions,
		s.idleTimeout,
		s.streamCreationTimeout,
		portforward.SupportedProtocols)
}
