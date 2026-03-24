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

//go:embed jobset_manifest.yaml.tpl
var jobsetManifestYamlTpl string

var jobsetManifestYamlTemplate = template.Must(template.New("jobset_manifest").Parse(jobsetManifestYamlTpl))

// BuildJobSetManifestConfig is the config for BuildJobSetManifest.
type BuildJobSetManifestConfig struct {
	Port         uint32
	ExternalName string
	CABundle     string
}

// BuildJobSetManifest builds the jobset manifest yaml content.
func BuildJobSetManifest(conf BuildJobSetManifestConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := jobsetManifestYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute jobset manifest yaml template: %w", err)
	}
	return buf.String(), nil
}
