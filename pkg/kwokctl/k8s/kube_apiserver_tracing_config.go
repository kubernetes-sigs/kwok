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

package k8s

import (
	"bytes"
	"fmt"
	"text/template"

	_ "embed"
)

//go:embed kube_apiserver_tracing_config.yaml.tpl
var kubeApiserverTracingConfigYamlTpl string

var kubeApiserverTracingConfigYamlTemplate = template.Must(template.New("kube_apiserver_tracing_config").Parse(kubeApiserverTracingConfigYamlTpl))

// BuildKubeApiserverTracingConfig builds a apiserverTracingConfig file from the given parameters.
func BuildKubeApiserverTracingConfig(conf BuildKubeApiserverTracingConfigParam) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kubeApiserverTracingConfigYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build apiserverTracingConfig error: %w", err)
	}
	return buf.String(), nil
}

// BuildKubeApiserverTracingConfigParam is the configuration for BuildKubeApiserverTracingConfig.
type BuildKubeApiserverTracingConfigParam struct {
	Endpoint string
}
