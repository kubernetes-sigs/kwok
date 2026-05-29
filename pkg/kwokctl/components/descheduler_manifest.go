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

// BuildDeschedulerManifestConfig is the config for BuildDeschedulerManifest.
type BuildDeschedulerManifestConfig struct {
	RawManifest string
}

// BuildDeschedulerManifest transforms raw descheduler manifest data with the provided
// configuration values. Resources that kwok does not run (Deployment, ConfigMap,
// ServiceAccount, RBAC, etc.) are removed from the manifest.
func BuildDeschedulerManifest(conf BuildDeschedulerManifestConfig) ([]string, error) {
	if len(conf.RawManifest) == 0 {
		return nil, fmt.Errorf("raw descheduler manifest is empty")
	}

	result, err := rewriteManifest(conf.RawManifest, defaultTransformers)
	if err != nil {
		return nil, fmt.Errorf("failed to transform descheduler manifest: %w", err)
	}
	return result, nil
}
