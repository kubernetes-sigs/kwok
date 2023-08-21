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

package kind

import (
	"bytes"
	"fmt"
	"text/template"

	_ "embed"
)

//go:embed external_metrics_apiservice.yaml.tpl
var externalMetricsApiserviceYamlTpl string

var externalMetricsApiserviceYamlTemplate = template.Must(template.New("external_metrics_apiservice").Parse(externalMetricsApiserviceYamlTpl))

// BuildExternalMetricsApiservice builds the externalMetricsApiservice yaml content.
func BuildExternalMetricsApiservice(conf BuildExternalMetricsApiserviceConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := externalMetricsApiserviceYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build externalMetricsApiservice error: %w", err)
	}
	return buf.String(), nil
}

// BuildExternalMetricsApiserviceConfig is the configuration for building the externalMetricsApiservice config
type BuildExternalMetricsApiserviceConfig struct {
	KwokControllerPort uint32
}
