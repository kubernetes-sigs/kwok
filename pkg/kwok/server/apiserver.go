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
	"strings"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/apiserver/pkg/endpoints/request"

	"sigs.k8s.io/kwok/pkg/kwok/apiserver"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

func newRequestInfoResolver() *request.RequestInfoFactory {
	return &request.RequestInfoFactory{
		APIPrefixes: sets.NewString(strings.Trim(discovery.APIGroupPrefix, "/")),
	}
}

// InstallAPIServer installs the API server for the server.
func (s *Server) InstallAPIServer(ctx context.Context) error {
	s.discoveryGroupManager = apiserver.InstallRootAPIs(s.restfulCont)
	return nil
}

// InstallOpenAPI installs the OpenAPI spec for the server.
func (s *Server) InstallOpenAPI(ctx context.Context) error {
	webServices := s.restfulCont.RegisteredWebServices()
	webServices = slices.Filter(webServices, func(r *restful.WebService) bool {
		return strings.HasPrefix(r.RootPath(), discovery.APIGroupPrefix+"/")
	})

	_, _, err := apiserver.InstallOpenAPIV2(s.restfulCont, webServices)
	if err != nil {
		return err
	}

	_, err = apiserver.InstallOpenAPIV3(s.restfulCont, webServices)
	if err != nil {
		return err
	}

	return nil
}

// InstallCustomMetricsAPI installs the custom metrics API for the server.
func (s *Server) InstallCustomMetricsAPI(ctx context.Context) error {
	customMetricsProvider := apiserver.NewCustomMetricsProvider(apiserver.CustomMetricsProviderConfig{
		Ctx:           ctx,
		DynamicClient: s.dynamicClient,
		RESTMapper:    s.restMapper,
		CustomMetric:  s.customMetrics,
	})
	err := customMetricsProvider.Install(s.restfulCont, s.discoveryGroupManager)
	if err != nil {
		return err
	}

	return nil
}

// InstallExternalMetricsAPI installs the external metrics API for the server.
func (s *Server) InstallExternalMetricsAPI(ctx context.Context) error {
	externalMetricsProvider := apiserver.NewExternalMetricsProvider(apiserver.ExternalMetricProviderConfig{
		ExternalMetric: s.externalMetrics,
	})
	err := externalMetricsProvider.Install(s.restfulCont, s.discoveryGroupManager)
	if err != nil {
		return err
	}

	return nil
}
