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
	"reflect"

	"github.com/emicklei/go-restful/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/util/flushwriter"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// GetContainerLogs returns logs for a container in a pod.
// If follow is true, it streams the logs until the connection is closed by the client.
func (s *Server) GetContainerLogs(ctx context.Context, podName, podNamespace, container string, logOptions *corev1.PodLogOptions, stdout, stderr io.Writer) error {
	// TODO: Configure and implement the log streamer
	msg := fmt.Sprintf("TODO: GetContainerLogs(%q, %q)", podName+"/"+podNamespace, container)

	log, err := s.getPodLogs(podName, podNamespace, container)
	if err != nil {
		return err
	}
	_ = log
	_, _ = stdout.Write([]byte(msg))
	return nil
}

// getContainerLogs handles containerLogs request against the Kubelet
func (s *Server) getContainerLogs(request *restful.Request, response *restful.Response) {
	podNamespace := request.PathParameter("podNamespace")
	podName := request.PathParameter("podID")
	containerName := request.PathParameter("containerName")

	if len(podName) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podID."}`))
		return
	}
	if len(containerName) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing restfulCont name."}`))
		return
	}
	if len(podNamespace) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podNamespace."}`))
		return
	}

	query := request.Request.URL.Query()
	// backwards compatibility for the "tail" query parameter
	if tail := request.QueryParameter("tail"); len(tail) > 0 {
		query["tailLines"] = []string{tail}
		// "all" is the same as omitting tail
		if tail == "all" {
			delete(query, "tailLines")
		}
	}

	// restfulCont logs on the kubelet are locked to the corev1 API version of PodLogOptions
	logOptions := &corev1.PodLogOptions{}
	err := convert_url_Values_To_v1_PodLogOptions(&query, logOptions, nil)
	if err != nil {
		logger := log.FromContext(request.Request.Context())
		logger.Error("Unable to decode the request for container logs", err)
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Unable to decode query."}`))
		return
	}

	if _, ok := response.ResponseWriter.(http.Flusher); !ok {
		_ = response.WriteError(http.StatusInternalServerError, fmt.Errorf("unable to convert %v into http.Flusher, cannot show logs", reflect.TypeOf(response)))
		return
	}
	fw := flushwriter.Wrap(response.ResponseWriter)
	response.Header().Set("Transfer-Encoding", "chunked")
	if err := s.GetContainerLogs(request.Request.Context(), podName, podNamespace, containerName, logOptions, fw, fw); err != nil {
		_ = response.WriteError(http.StatusBadRequest, err)
		return
	}
}

func (s *Server) getPodLogs(podName, podNamespace, container string) (*internalversion.Log, error) {
	l, has := slices.Find(s.config.Logs, func(l *internalversion.Logs) bool {
		return l.Name == podName && l.Namespace == podNamespace
	})
	if has {
		log, found := findLogInLogs(container, l.Spec.Logs)
		if found {
			return log, nil
		}
	} else {
		for _, cl := range s.config.ClusterLogs {
			if !cl.Spec.Selector.Match(podName, podNamespace) {
				continue
			}

			log, found := findLogInLogs(container, cl.Spec.Logs)
			if found {
				return log, nil
			}
		}
	}
	return nil, fmt.Errorf("Failed to find pod with container name %q", container)
}

func findLogInLogs(container string, logs []internalversion.Log) (*internalversion.Log, bool) {
	var defaultLog *internalversion.Log
	for i, l := range logs {
		if len(l.Containers) == 0 && defaultLog == nil {
			defaultLog = &logs[i]
			continue
		}
		if len(container) != 0 {
			if slices.Contains(l.Containers, container) {
				return &l, true
			}
		}
	}
	return defaultLog, defaultLog != nil
}
