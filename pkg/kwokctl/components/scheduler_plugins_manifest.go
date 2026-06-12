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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BuildSchedulerPluginsManifestConfig is the configuration for building the scheduler-plugins manifest.
type BuildSchedulerPluginsManifestConfig struct {
	RawManifest string
}

// BuildSchedulerPluginsManifest transforms raw scheduler-plugins manifest data with the provided
func BuildSchedulerPluginsManifest(conf BuildSchedulerPluginsManifestConfig) ([]string, error) {
	if len(conf.RawManifest) == 0 {
		return nil, fmt.Errorf("raw scheduler-plugins manifest is empty")
	}

	transformers := append([]resourceTransformer{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
			Match: func(obj *unstructured.Unstructured) bool {
				return obj.GetName() == "system:kube-scheduler:plugins"
			},
		},
		{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
			Match: func(obj *unstructured.Unstructured) bool {
				return obj.GetName() == "system:kube-scheduler:plugins"
			},
		},
	}, defaultTransformers...)

	result, err := rewriteManifest(conf.RawManifest, transformers)
	if err != nil {
		return nil, fmt.Errorf("failed to transform scheduler-plugins manifest: %w", err)
	}
	return result, nil
}
