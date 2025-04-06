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
	"time"

	"github.com/emicklei/go-restful/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/util/flushwriter"
	criapi "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	crilogs "k8s.io/cri-client/pkg/logs"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// GetContainerLogs returns logs for a container in a pod.
// If follow is true, it streams the logs until the connection is closed by the client.
func (s *Server) GetContainerLogs(ctx context.Context, podName, podNamespace, container string, logOptions *corev1.PodLogOptions, stdout, stderr io.Writer) error {
	log, err := getPodLogs(s.logs.Get(), s.clusterLogs.Get(), podName, podNamespace, container)
	if err != nil {
		return err
	}

	opts := crilogs.NewLogOptions(logOptions, time.Now())
	logsFile := log.LogsFile
	if logOptions.Previous {
		logsFile = log.PreviousLogsFile
	}
	return readLogs(ctx, logsFile, opts, stdout, stderr)
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

func getPodLogs(rules []*internalversion.Logs, clusterRules []*internalversion.ClusterLogs, podName, podNamespace, containerName string) (*internalversion.Log, error) {
	l, has := slices.Find(rules, func(l *internalversion.Logs) bool {
		return l.Name == podName && l.Namespace == podNamespace
	})
	if has {
		l, found := findLogInLogs(containerName, l.Spec.Logs)
		if found {
			return l, nil
		}
		return nil, fmt.Errorf("log target not found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
	}

	for _, cl := range clusterRules {
		if !cl.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		log, found := findLogInLogs(containerName, cl.Spec.Logs)
		if found {
			return log, nil
		}
	}
	return nil, fmt.Errorf("no logs found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
}

func findLogInLogs(containerName string, logs []internalversion.Log) (*internalversion.Log, bool) {
	var defaultLog *internalversion.Log
	for i, l := range logs {
		if len(l.Containers) == 0 && defaultLog == nil {
			defaultLog = &logs[i]
			continue
		}
		if slices.Contains(l.Containers, containerName) {
			return &l, true
		}
	}
	return defaultLog, defaultLog != nil
}

func readLogs(ctx context.Context, logsFile string, opts *crilogs.LogOptions, stdout, stderr io.Writer) error {
	return crilogs.ReadLogs(ctx, nil, logsFile, "", opts, runtimeServiceStub{}, stdout, stderr)
}

type runtimeServiceStub struct {
	criapi.RuntimeService
}

var errUnavailable = status.Error(codes.Unavailable, "Unavailable")

func (runtimeServiceStub) ContainerStatus(ctx context.Context, containerID string, verbose bool) (*runtimeapi.ContainerStatusResponse, error) {
	return nil, errUnavailable
}
