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

package binary

import (
	"bytes"
	"fmt"
	"text/template"

	_ "embed"
)

//go:embed kwok_service.yaml.tpl
var kwokServiceYamlTpl string

var kwokServiceYamlTemplate = template.Must(template.New("kwok_service").Parse(kwokServiceYamlTpl))

// BuildKwokService builds the kwokService yaml content.
func BuildKwokService(conf BuildKwokServiceConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kwokServiceYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build kwokService error: %w", err)
	}
	return buf.String(), nil
}

// BuildKwokServiceConfig is the configuration for building the kwokService config
type BuildKwokServiceConfig struct {
}
