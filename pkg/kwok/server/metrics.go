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
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/kwok/metrics"
	"sigs.k8s.io/kwok/pkg/kwok/metrics/cel"
	"sigs.k8s.io/kwok/pkg/log"
)

// InstallMetrics registers the metrics handler on the given mux.
func (s *Server) InstallMetrics(ctx context.Context) error {
	promHandler := promhttp.Handler()

	selfMetric := func(req *restful.Request, resp *restful.Response) {
		promHandler.ServeHTTP(resp.ResponseWriter, req.Request)
	}

	controller := s.controller
	env, err := cel.NewEnvironment(cel.NodeEvaluatorConfig{
		EnableEvaluatorCache: true,
		EnableResultCache:    true,
		StartedContainersTotal: func(nodeName string) int64 {
			nodeInfo, ok := controller.GetNodeInfo(nodeName)
			if !ok {
				return 0
			}
			return nodeInfo.StartedContainer.Load()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create CEL environment: %w", err)
	}

	const rootPath = "/metrics"
	ws := new(restful.WebService)
	ws.Path(rootPath)
	ws.Route(ws.GET("/").To(selfMetric))

	for _, m := range s.metrics.Get() {
		if !strings.HasPrefix(m.Spec.Path, rootPath) {
			return fmt.Errorf("metric path %q does not start with %q", m.Spec.Path, rootPath)
		}

		ws.Route(ws.GET(strings.TrimPrefix(m.Spec.Path, rootPath)).
			To(s.getMetrics(m, env)))
	}

	s.restfulCont.Add(ws)
	s.metricsWebService = ws

	logger := log.FromContext(ctx)
	syncd, ok := s.metrics.(resources.Synced)
	if ok {
		logger.Info("Starting metrics syncer")
		go func() {
			for range syncd.Sync() {
				logger.Info("Metrics synced, updating metrics web service")
				ws := new(restful.WebService)
				ws.Path(rootPath)
				ws.Route(ws.GET("/").To(selfMetric))

				for _, m := range s.metrics.Get() {
					if !strings.HasPrefix(m.Spec.Path, rootPath) {
						logger.Warn("metric path does not start with "+rootPath, "path", m.Spec.Path)
						continue
					}
					ws.Route(ws.GET(strings.TrimPrefix(m.Spec.Path, rootPath)).
						To(s.getMetrics(m, env)))
				}

				err := s.restfulCont.Remove(s.metricsWebService)
				if err != nil {
					logger.Error("failed to remove metrics web service", err)
				}
				s.restfulCont.Add(ws)
				s.metricsWebService = ws
			}
		}()
	}

	return nil
}

func (s *Server) getMetrics(metric *internalversion.Metric, env *cel.Environment) func(req *restful.Request, resp *restful.Response) {
	return func(req *restful.Request, resp *restful.Response) {
		nodeName := req.PathParameter("nodeName")
		if nodeName == "" {
			nodeName = metric.Name
		}

		handler, ok := s.metricsUpdateHandler.Load(nodeName)
		if !ok {
			handler = metrics.NewMetricsUpdateHandler(metrics.UpdateHandlerConfig{
				Controller:  s.controller,
				Environment: env,
			})
			s.metricsUpdateHandler.Store(nodeName, handler)
		}

		handler.Update(req.Request.Context(), nodeName, metric.Spec.Metrics)
		handler.ServeHTTP(resp.ResponseWriter, req.Request)
	}
}
