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

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sigs.k8s.io/kwok/pkg/kwok/metrics"
	"sigs.k8s.io/kwok/pkg/kwok/metrics/cel"
)

// InstallMetrics registers the metrics handler on the given mux.
func (s *Server) InstallMetrics() error {
	promHandler := promhttp.Handler()
	s.restfulCont.Handle("/metrics", promHandler)

	controller := s.config.Controller
	env, err := cel.NewEnvironment(cel.NodeEvaluatorConfig{
		StartedContainersTotal: func(nodeName string) int64 {
			nodeInfo, ok := controller.GetNode(nodeName)
			if !ok {
				return 0
			}
			return nodeInfo.StartedContainer.Load()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create CEL environment: %w", err)
	}
	for _, m := range s.config.Metrics {
		handler, err := metrics.NewMetricsUpdateHandler(metrics.UpdateHandlerConfig{
			NodeName:    m.Name,
			Metrics:     m,
			Controller:  controller,
			Environment: env,
		})
		if err != nil {
			return fmt.Errorf("failed to create metrics update handler: %w", err)
		}
		s.restfulCont.Handle(m.Spec.Path, handler)
	}
	return nil
}
