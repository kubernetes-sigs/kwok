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
	"strings"
	"text/template"

	_ "embed"
)

//go:embed kwok_controller_pod.yaml.tpl
var kwokControllerPodYamlTpl string

var kwokControllerPodYamlTemplate = template.Must(template.New("_").Parse(kwokControllerPodYamlTpl))

func BuildKwokControllerPod(conf BuildKwokControllerPodConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	split := strings.SplitN(conf.KwokControllerImage, ":", 2)
	conf.KwokControllerImageName = split[0]
	conf.KwokControllerImageTag = split[1]
	err := kwokControllerPodYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute kwok controller pod yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildKwokControllerPodConfig struct {
	KwokControllerImage     string
	KwokControllerImageName string
	KwokControllerImageTag  string
	Name                    string
}
