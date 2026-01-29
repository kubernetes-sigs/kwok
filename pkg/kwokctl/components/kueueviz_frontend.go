/*
Copyright 2026 The Kubernetes Authors.

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

package components

import (
	"fmt"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

// BuildKueuevizFrontendComponentConfig is the configuration for building a kueueviz frontend component.
type BuildKueuevizFrontendComponentConfig struct {
	Runtime     string
	ProjectName string
	Binary      string
	Image       string

	BackendPort uint32
	Port        uint32
}

// BuildKueuevizFrontendComponent builds a kueueviz frontend component.
func BuildKueuevizFrontendComponent(conf BuildKueuevizFrontendComponentConfig) (component internalversion.Component, err error) {
	if GetRuntimeMode(conf.Runtime) != RuntimeModeContainer {
		return internalversion.Component{}, fmt.Errorf("kueue only supports container runtime for now")
	}

	var ports []internalversion.Port
	var metric *internalversion.ComponentMetric

	envs := []internalversion.Env{
		{
			Name:  "REACT_APP_WEBSOCKET_URL",
			Value: "ws://localhost:" + format.String(conf.BackendPort),
		},
	}

	ports = append(
		ports,
		internalversion.Port{
			Name:     "http",
			HostPort: conf.Port,
			Port:     8080,
			Protocol: internalversion.ProtocolTCP,
		},
	)

	return internalversion.Component{
		Name: consts.ComponentKueueviz + "-frontend",
		Links: []string{
			consts.ComponentKueueviz + "-backend",
		},
		Binary: conf.Binary,
		Image:  conf.Image,
		Ports:  ports,
		Metric: metric,
		Envs:   envs,
	}, nil
}
