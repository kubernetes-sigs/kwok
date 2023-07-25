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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"

	_ "embed"
)

//go:embed metrics_server_deployment.yaml.tpl
var metricsServerDeploymentYamlTpl string

var metricsServerDeploymentYamlTemplate = template.Must(template.New("_").Parse(metricsServerDeploymentYamlTpl))

// BuildMetricsServerDeployment builds the metrics server deployment yaml content.
func BuildMetricsServerDeployment(conf BuildMetricsServerDeploymentConfig) (string, error) {
	buf := bytes.NewBuffer(nil)

	var err error
	conf.ExtraVolumes, err = runtime.ExpandVolumesHostPaths(conf.ExtraVolumes)
	if err != nil {
		return "", fmt.Errorf("failed to expand host volume paths: %w", err)
	}

	err = metricsServerDeploymentYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute metrics server deployment yaml template: %w", err)
	}
	return buf.String(), nil
}

// BuildMetricsServerDeploymentConfig is the configuration for building the metrics server deployment
type BuildMetricsServerDeploymentConfig struct {
	MetricsServerImage string
	Name               string
	Verbosity          int
	ExtraArgs          []internalversion.ExtraArgs
	ExtraVolumes       []internalversion.Volume
	ExtraEnvs          []internalversion.Env
}
