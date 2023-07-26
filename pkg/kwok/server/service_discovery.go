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
	"encoding/json"
	"net/http"
	"strings"

	"sigs.k8s.io/kwok/pkg/kwok/controllers"
)

// InstallServiceDiscovery installs the service discovery handler.
func (s *Server) InstallServiceDiscovery() {
	s.restfulCont.Handle("/discovery/prometheus", http.HandlerFunc(s.prometheusDiscovery))
}

func (s *Server) prometheusDiscovery(rw http.ResponseWriter, req *http.Request) {
	targets := []prometheusStaticConfig{}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	hosts := []string{req.Host}

	var listNode []*controllers.NodeInfo

	metrics := s.metrics.Get()
	for _, m := range metrics {
		if strings.Contains(m.Spec.Path, "{nodeName}") {
			if listNode == nil {
				listNode = s.controller.ListNodes()
			}
			for _, node := range listNode {
				targets = append(targets, prometheusStaticConfig{
					Targets: hosts,
					Labels: map[string]string{
						"metrics_name":     m.Name,
						"__scheme__":       scheme,
						"__metrics_path__": strings.ReplaceAll(m.Spec.Path, "{nodeName}", node.Node.Name),
					},
				})
			}
		} else {
			targets = append(targets, prometheusStaticConfig{
				Targets: hosts,
				Labels: map[string]string{
					"metrics_name":     m.Name,
					"__scheme__":       scheme,
					"__metrics_path__": m.Spec.Path,
				},
			})
		}
	}
	rw.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(rw).Encode(targets)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

type prometheusStaticConfig struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
}
