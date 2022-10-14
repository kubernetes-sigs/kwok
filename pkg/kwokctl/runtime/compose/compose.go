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

package compose

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed compose.yaml.tpl
var composeYamlTpl string

var composeYamlTemplate = template.Must(template.New("_").Parse(composeYamlTpl))

func BuildCompose(conf BuildComposeConfig) (string, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYamlTemplate.Execute(buf, conf)
	if err != nil {
		return "", fmt.Errorf("failed to execute compose yaml template: %w", err)
	}
	return buf.String(), nil
}

type BuildComposeConfig struct {
	ProjectName string

	PrometheusImage            string
	EtcdImage                  string
	KubeApiserverImage         string
	KubeControllerManagerImage string
	KubeSchedulerImage         string
	KwokControllerImage        string

	PrometheusPath          string
	AdminKeyPath            string
	AdminCertPath           string
	CACertPath              string
	KubeconfigPath          string
	InClusterAdminKeyPath   string
	InClusterAdminCertPath  string
	InClusterCACertPath     string
	InClusterKubeconfigPath string
	InClusterEtcdDataPath   string
	InClusterPrometheusPath string

	SecretPort bool
	QuietPull  bool

	KubeApiserverPort uint32
	PrometheusPort    uint32

	RuntimeConfig string
	FeatureGates  string

	AuditPolicy string
	AuditLog    string
}
