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

package components

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"

	_ "embed"
)

//go:embed prometheus_config.yaml.tpl
var prometheusYamlTpl string

var prometheusYamlTemplate = template.Must(template.New("prometheus_config").Funcs(sprig.TxtFuncMap()).Parse(prometheusYamlTpl))

// BuildPrometheus builds the prometheus yaml content.
func BuildPrometheus(conf BuildPrometheusConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := prometheusYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build prometheus error: %w", err)
	}
	return buf.String(), nil
}

// BuildPrometheusConfig is the configuration for building the prometheus config
type BuildPrometheusConfig struct {
	Components []internalversion.Component
}
