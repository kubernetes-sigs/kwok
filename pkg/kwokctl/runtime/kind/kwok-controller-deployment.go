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
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

//go:generate sh -c "kubectl kustomize . > kwok-controller-deployment.yaml.tpl"

//go:embed kwok-controller-deployment.yaml.tpl
var kwokControllerDeploymentYamlTpl string

var kwokControllerDeploymentYamlTemplate = template.Must(template.New("_").Parse(kwokControllerDeploymentYamlTpl))

func BuildKwokControllerDeploy(conf BuildKwokControllerDeployConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	split := strings.SplitN(conf.KwokControllerImage, ":", 2)
	conf.KwokControllerImageName = split[0]
	conf.KwokControllerImageTag = split[1]
	err := kwokControllerDeploymentYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kwok controller deploy yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildKwokControllerDeployConfig struct {
	KwokControllerImage     string
	KwokControllerImageName string
	KwokControllerImageTag  string
	Name                    string
}
