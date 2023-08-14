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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"

	_ "embed"
)

//go:embed prometheus_deployment.yaml.tpl
var prometheusDeploymentYamlTpl string

var prometheusDeploymentYamlTemplate = template.Must(template.New("prometheus_deployment").Parse(prometheusDeploymentYamlTpl))

// BuildPrometheusDeployment builds the prometheus deployment yaml content.
func BuildPrometheusDeployment(conf BuildPrometheusDeploymentConfig) (string, error) {
	buf := bytes.NewBuffer(nil)

	var err error
	conf.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.ExtraVolumes)
	if err != nil {
		return "", fmt.Errorf("failed to expand host volume paths: %w", err)
	}

	err = prometheusDeploymentYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute prometheus deployment yaml template: %w", err)
	}
	return buf.String(), nil
}

// BuildPrometheusDeploymentConfig is the configuration for building the prometheus deployment
type BuildPrometheusDeploymentConfig struct {
	PrometheusImage string
	Name            string
	LogLevel        string
	ExtraArgs       []internalversion.ExtraArgs
	ExtraVolumes    []internalversion.Volume
	ExtraEnvs       []internalversion.Env
}
