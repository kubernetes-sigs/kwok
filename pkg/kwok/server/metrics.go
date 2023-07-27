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
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/kwok/metrics"
	"sigs.k8s.io/kwok/pkg/kwok/metrics/cel"
	"sigs.k8s.io/kwok/pkg/log"
)

func (s *Server) initCEL() error {
	if s.env != nil {
		return fmt.Errorf("CEL environment already initialized")
	}

	env, err := cel.NewEnvironment(cel.NodeEvaluatorConfig{
		EnableEvaluatorCache:   true,
		EnableResultCache:      true,
		StartedContainersTotal: s.dataSource.StartedContainersTotal,

		ContainerResourceUsage: s.containerResourceUsage,
		PodResourceUsage:       s.podResourceUsage,
		NodeResourceUsage:      s.nodeResourceUsage,

		ContainerResourceCumulativeUsage: s.containerResourceCumulativeUsage,
		PodResourceCumulativeUsage:       s.podResourceCumulativeUsage,
		NodeResourceCumulativeUsage:      s.nodeResourceCumulativeUsage,
	})
	if err != nil {
		return fmt.Errorf("failed to create CEL environment: %w", err)
	}
	s.env = env
	return nil
}

// InstallMetrics registers the metrics handler on the given mux.
func (s *Server) InstallMetrics(ctx context.Context) error {
	err := s.initCEL()
	if err != nil {
		return err
	}

	promHandler := promhttp.Handler()

	selfMetric := func(req *restful.Request, resp *restful.Response) {
		promHandler.ServeHTTP(resp.ResponseWriter, req.Request)
	}

	const rootPath = "/metrics"
	ws := new(restful.WebService)
	ws.Path(rootPath)
	ws.Route(ws.GET("/").To(selfMetric))
	s.restfulCont.Add(ws)

	hasPaths := map[string]struct{}{}
	logger := log.FromContext(ctx)
	syncd, ok := s.metrics.(resources.Synced)
	if ok {
		go func() {
			for range syncd.Sync() {
				newHasPaths := map[string]struct{}{}
				for _, m := range s.metrics.Get() {
					if !strings.HasPrefix(m.Spec.Path, rootPath) {
						logger.Warn("metric path does not start with "+rootPath, "path", m.Spec.Path)
						continue
					}

					path := strings.TrimPrefix(m.Spec.Path, rootPath)
					newHasPaths[path] = struct{}{}
					if _, ok := hasPaths[path]; ok {
						err := ws.RemoveRoute(http.MethodGet, path)
						if err != nil {
							logger.Error("Failed to remove route", err, "path", path)
						}
					}
					ws.Route(ws.GET(path).
						To(s.getMetrics(m, s.env)))
				}

				for path := range hasPaths {
					if _, ok := newHasPaths[path]; !ok {
						err := ws.RemoveRoute(http.MethodGet, path)
						if err != nil {
							logger.Error("Failed to remove route", err, "path", path)
						}
					}
				}

				hasPaths = newHasPaths
			}
		}()
	} else {
		for _, m := range s.metrics.Get() {
			if !strings.HasPrefix(m.Spec.Path, rootPath) {
				return fmt.Errorf("metric path %q does not start with %q", m.Spec.Path, rootPath)
			}
			ws.Route(ws.GET(strings.TrimPrefix(m.Spec.Path, rootPath)).
				To(s.getMetrics(m, s.env)))
		}
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
				Environment:     env,
				DataSource:      s.dataSource,
				NodeCacheGetter: s.nodeCacheGetter,
				PodCacheGetter:  s.podCacheGetter,
			})
			s.metricsUpdateHandler.Store(nodeName, handler)
		}

		handler.Update(req.Request.Context(), nodeName, metric.Spec.Metrics)
		handler.ServeHTTP(resp.ResponseWriter, req.Request)
	}
}
