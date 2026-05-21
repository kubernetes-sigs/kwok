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

const lwsConfigMapName = "lws-manager-config"

// BuildLWSConfig builds the lws configuration from the upstream manifest.
func BuildLWSConfig(rawManifest string) (string, error) {
	if rawManifest == "" {
		return "", fmt.Errorf("raw lws manifest is empty")
	}

	rawConfig, err := getConfigFromManifest(rawManifest, lwsConfigMapName, controllerManagerConfigKey)
	if err != nil {
		return "", fmt.Errorf("get config from manifest: %w", err)
	}

	config, err := rewriteConfig(rawConfig, func(config map[string]any) error {
		if err := unstructured.SetNestedField(config, false, "leaderElection", "leaderElect"); err != nil {
			return fmt.Errorf("set leaderElection.leaderElect: %w", err)
		}
		if err := unstructured.SetNestedField(config, false, "internalCertManagement", "enable"); err != nil {
			return fmt.Errorf("set internalCertManagement.enable: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("rewrite config: %w", err)
	}

	return config, nil
}
