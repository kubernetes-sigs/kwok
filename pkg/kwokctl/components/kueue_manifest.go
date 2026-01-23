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
	"bytes"
	"fmt"
	"text/template"

	_ "embed"
)

//go:embed kueue_manifest.yaml.tpl
var kueueManifestYamlTpl string

var kueueManifestYamlTemplate = template.Must(template.New("kueue_manifest").Parse(kueueManifestYamlTpl))

// BuildKueueManifest builds the kueue webhook yaml content.
func BuildKueueManifest(conf BuildManifestConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kueueManifestYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kueue webhook yaml template: %w", err)
	}
	return buf.String(), nil
}

// BuildManifestConfig is the config for BuildKueueWebhook.
type BuildManifestConfig struct {
	Port           uint32
	ExternalName   string
	VisibilityPort uint32
	CABundle       string
}
