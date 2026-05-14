/*
Copyright 2024 The Kubernetes Authors.

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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BuildMetricsServerManifestConfig is the config for BuildMetricsServerManifest.
type BuildMetricsServerManifestConfig struct {
	Port         uint32
	ExternalName string
	RawManifest  string
}

// BuildMetricsServerManifest transforms raw metrics-server manifest data with
// the provided configuration values. The metrics-server Deployment is removed
// since kwok does not run it.
func BuildMetricsServerManifest(conf BuildMetricsServerManifestConfig) ([]string, error) {
	if len(conf.RawManifest) == 0 {
		return nil, fmt.Errorf("raw metrics-server manifest is empty")
	}

	transformers := append([]resourceTransformer{
		{
			Kind:       "APIService",
			APIVersion: "apiregistration.k8s.io/v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformAPIService(obj.Object, int64(conf.Port))
			},
		},
		{
			Kind:       "Service",
			APIVersion: "v1",
			Transform: func(obj *unstructured.Unstructured) error {
				return transformServiceToExternalName(obj.Object, conf.ExternalName)
			},
		},
	}, defaultTransformers...)

	result, err := rewriteManifest(conf.RawManifest, transformers)
	if err != nil {
		return nil, fmt.Errorf("failed to transform metrics-server manifest: %w", err)
	}
	return result, nil
}
