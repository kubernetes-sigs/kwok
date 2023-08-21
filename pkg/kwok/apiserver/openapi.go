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

package apiserver

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful/v3"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/builder3"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/common/restfuladapter"
	"k8s.io/kube-openapi/pkg/handler"
	"k8s.io/kube-openapi/pkg/handler3"
	"k8s.io/kube-openapi/pkg/validation/spec"
	generatedcore "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/core"
	generatedcustommetrics "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/custommetrics"
	generatedexternalmetrics "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/externalmetrics"
)

// InstallOpenAPIV2 adds the SwaggerUI webservice to the given mux.
func InstallOpenAPIV2(container *restful.Container, webServices []*restful.WebService) (*handler.OpenAPIService, *spec.Swagger, error) {
	conf := openAPIConfig(defaultOpenAPIConfig)
	spec, err := builder.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(webServices), conf)
	if err != nil {
		return nil, nil, fmt.Errorf("error building openapi v2 spec: %w", err)
	}
	spec.Definitions = handler.PruneDefaults(spec.Definitions)
	openAPIVersionedService := handler.NewOpenAPIService(spec)
	openAPIVersionedService.RegisterOpenAPIVersionedService("/openapi/v2", container)

	return openAPIVersionedService, spec, nil
}

// InstallOpenAPIV3 adds the static group/versions defined in the RegisteredWebServices to the OpenAPI v3 spec
func InstallOpenAPIV3(container *restful.Container, webServices []*restful.WebService) (*handler3.OpenAPIService, error) {
	conf := openAPIConfig(defaultOpenAPIV3Config)
	openAPIVersionedService := handler3.NewOpenAPIService()
	container.Handle("/openapi/v3", http.HandlerFunc(openAPIVersionedService.HandleDiscovery))
	container.Handle("/openapi/v3/", http.HandlerFunc(openAPIVersionedService.HandleGroupVersion))

	grouped := make(map[string][]*restful.WebService)

	for _, t := range webServices {
		// Strip the "/" prefix from the name
		gvName := t.RootPath()[1:]
		grouped[gvName] = []*restful.WebService{t}
	}

	for gv, ws := range grouped {
		spec, err := builder3.BuildOpenAPISpecFromRoutes(restfuladapter.AdaptWebServices(ws), conf)
		if err != nil {
			return nil, fmt.Errorf("error building openapi v3 spec: %w", err)
		}
		openAPIVersionedService.UpdateGroupVersion(gv, spec)
	}
	return openAPIVersionedService, nil
}

func mergeOpenAPIDefinitions(definitionsGetters []openapicommon.GetOpenAPIDefinitions) openapicommon.GetOpenAPIDefinitions {
	return func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		defsMap := make(map[string]openapicommon.OpenAPIDefinition)
		for _, definitionsGetter := range definitionsGetters {
			definitions := definitionsGetter(ref)
			for k, v := range definitions {
				defsMap[k] = v
			}
		}
		return defsMap
	}
}

func openAPIConfig(createConfig func(getDefinitions openapicommon.GetOpenAPIDefinitions, defNamer *openapi.DefinitionNamer) *openapicommon.Config) *openapicommon.Config {
	definitionsGetters := []openapicommon.GetOpenAPIDefinitions{generatedcore.GetOpenAPIDefinitions}

	definitionsGetters = append(definitionsGetters, generatedcustommetrics.GetOpenAPIDefinitions)

	definitionsGetters = append(definitionsGetters, generatedexternalmetrics.GetOpenAPIDefinitions)

	getAPIDefinitions := mergeOpenAPIDefinitions(definitionsGetters)
	openAPIConfig := createConfig(getAPIDefinitions, openapi.NewDefinitionNamer(scheme))
	openAPIConfig.Info.Title = "kwok-metrics"
	openAPIConfig.Info.Version = "1.0.0"
	return openAPIConfig
}

// defaultOpenAPIConfig provides the default OpenAPIConfig used to build the OpenAPI V2 spec
func defaultOpenAPIConfig(getDefinitions openapicommon.GetOpenAPIDefinitions, defNamer *openapi.DefinitionNamer) *openapicommon.Config {
	return &openapicommon.Config{
		ProtocolList:   []string{"https"},
		IgnorePrefixes: []string{},
		Info: &spec.Info{
			InfoProps: spec.InfoProps{
				Title: "Generic API Server",
			},
		},
		DefaultResponse: &spec.Response{
			ResponseProps: spec.ResponseProps{
				Description: "Default Response.",
			},
		},
		GetOperationIDAndTagsFromRoute: func(route openapicommon.Route) (string, []string, error) {
			restfulRouteAdapter, ok := route.(*restfuladapter.RouteAdapter)
			if !ok {
				return "", nil, fmt.Errorf("unexpected route type: %T", route)
			}
			return openapi.GetOperationIDAndTags(restfulRouteAdapter.Route)
		},
		GetDefinitionName: defNamer.GetDefinitionName,
		GetDefinitions:    getDefinitions,
	}
}

// defaultOpenAPIV3Config provides the default OpenAPIV3Config used to build the OpenAPI V3 spec
func defaultOpenAPIV3Config(getDefinitions openapicommon.GetOpenAPIDefinitions, defNamer *openapi.DefinitionNamer) *openapicommon.Config {
	defaultConfig := defaultOpenAPIConfig(getDefinitions, defNamer)
	defaultConfig.Definitions = getDefinitions(func(name string) spec.Ref {
		defName, _ := defaultConfig.GetDefinitionName(name)
		return spec.MustCreateRef("#/components/schemas/" + openapicommon.EscapeJsonPointer(defName))
	})

	return defaultConfig
}
