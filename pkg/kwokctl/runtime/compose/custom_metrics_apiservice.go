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

package compose

import (
	"bytes"
	"fmt"
	"text/template"

	_ "embed"
)

//go:embed custom_metrics_apiservice.yaml.tpl
var customMetricsApiserviceYamlTpl string

var customMetricsApiserviceYamlTemplate = template.Must(template.New("custom_metrics_apiservice").Parse(customMetricsApiserviceYamlTpl))

// BuildCustomMetricsApiservice builds the customMetricsApiservice yaml content.
func BuildCustomMetricsApiservice(conf BuildCustomMetricsApiserviceConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := customMetricsApiserviceYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build customMetricsApiservice error: %w", err)
	}
	return buf.String(), nil
}

// BuildCustomMetricsApiserviceConfig is the configuration for building the customMetricsApiservice config
type BuildCustomMetricsApiserviceConfig struct {
	KwokControllerPort uint32
}
