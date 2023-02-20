/*
Copyright 2022 The Kubernetes Authors.

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

//go:embed prometheus_deployment.yaml.tpl
var prometheusDeploymentYamlTpl string

var prometheusDeploymentYamlTemplate = template.Must(template.New("_").Parse(prometheusDeploymentYamlTpl))

// BuildPrometheusDeployment builds the prometheus deployment yaml content.
func BuildPrometheusDeployment(conf BuildPrometheusDeploymentConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := prometheusDeploymentYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute prometheus deployment yaml template: %w", err)
	}
	return buf.String(), nil
}

// BuildPrometheusDeploymentConfig is the configuration for building the prometheus deployment
type BuildPrometheusDeploymentConfig struct {
	PrometheusImage string
	Name            string
	Verbosity       int
	HumanVerbosity  string
}
