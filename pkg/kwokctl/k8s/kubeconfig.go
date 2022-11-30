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

package k8s

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed kubeconfig.yaml.tpl
var kubeconfigYamlTpl string

var kubeconfigYamlTemplate = template.Must(template.New("_").Parse(kubeconfigYamlTpl))

func BuildKubeconfig(conf BuildKubeconfigConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := kubeconfigYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("build kubeconfig error: %w", err)
	}
	return buf.String(), nil
}

type BuildKubeconfigConfig struct {
	ProjectName  string
	SecretPort   bool
	Address      string
	AdminCrtPath string
	AdminKeyPath string
}
