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
)

// BuildNodeReadinessControllerManifestConfig is the config for BuildNodeReadinessControllerManifest.
type BuildNodeReadinessControllerManifestConfig struct {
	RawManifest string
}

// BuildNodeReadinessControllerManifest transforms raw node-readiness-controller manifest data with the provided
// configuration values. Resources that kwok does not run (Deployment, ConfigMap,
// ServiceAccount, RBAC, etc.) are removed from the manifest.
func BuildNodeReadinessControllerManifest(conf BuildNodeReadinessControllerManifestConfig) ([]string, error) {
	if len(conf.RawManifest) == 0 {
		return nil, fmt.Errorf("raw node-readiness-controller manifest is empty")
	}

	transformers := append([]resourceTransformer{
		{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
	}, defaultTransformers...)

	result, err := rewriteManifest(conf.RawManifest, transformers)
	if err != nil {
		return nil, fmt.Errorf("failed to transform node-readiness-controller manifest: %w", err)
	}
	return result, nil
}
