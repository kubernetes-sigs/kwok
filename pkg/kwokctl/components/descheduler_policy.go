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

const deschedulerConfigMapName = "descheduler-policy-configmap"
const deschedulerPolicyKey = "policy.yaml"

// BuildDeschedulerPolicy extracts the descheduler policy from the upstream manifest.
func BuildDeschedulerPolicy(rawManifests []string) (string, error) {
	if len(rawManifests) == 0 {
		return "", fmt.Errorf("raw descheduler manifests are empty")
	}

	var rawConfig string
	for _, rawManifest := range rawManifests {
		config, err := getConfigFromManifest(rawManifest, deschedulerConfigMapName, deschedulerPolicyKey)
		if err != nil {
			return "", fmt.Errorf("get config from manifest: %w", err)
		}
		if config != "" {
			rawConfig = config
			break
		}
	}

	if rawConfig == "" {
		return "", fmt.Errorf("config not found in manifests")
	}

	return rawConfig, nil
}
