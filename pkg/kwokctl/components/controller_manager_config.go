/*
Copyright 2026 The Kubernetes Authors.

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
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	utilyaml "sigs.k8s.io/kwok/pkg/utils/yaml"
)

const controllerManagerConfigKey = "controller_manager_config.yaml"

func getConfigFromManifest(rawManifest, configMapName string, key string) (string, error) {
	if len(rawManifest) == 0 {
		return "", fmt.Errorf("raw manifest is empty")
	}

	reader := utilyaml.NewDecoder(bufio.NewReader(strings.NewReader(rawManifest)))
	for {
		obj, err := reader.DecodeUnstructured()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read yaml document: %w", err)
		}

		if obj.GetKind() != "ConfigMap" || obj.GetName() != configMapName {
			continue
		}

		data, found, err := unstructured.NestedStringMap(obj.Object, "data")
		if err != nil {
			return "", fmt.Errorf("configmap %q data: %w", configMapName, err)
		}
		if !found {
			return "", fmt.Errorf("configmap %q data not found", configMapName)
		}

		configData, ok := data[key]
		if !ok || configData == "" {
			return "", fmt.Errorf("configmap %q missing %q", configMapName, key)
		}

		return configData, nil
	}

	return "", fmt.Errorf("configmap %q not found in manifest", configMapName)
}

func rewriteConfig(config string, modifyFunc func(map[string]any) error) (string, error) {
	var configMap map[string]any
	if err := utilyaml.Unmarshal([]byte(config), &configMap); err != nil {
		return "", fmt.Errorf("unmarshal config: %w", err)
	}

	if err := modifyFunc(configMap); err != nil {
		return "", fmt.Errorf("modify config: %w", err)
	}

	modifiedConfigBytes, err := utilyaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("marshal modified config: %w", err)
	}

	return string(modifiedConfigBytes), nil
}
