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

package runtime

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// CopySchedulerConfig copies the scheduler configuration file to the given path.
func (c *Cluster) CopySchedulerConfig(oldpath, newpath, kubeconfig string) error {
	data, err := os.ReadFile(oldpath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", oldpath, err)
	}

	var config unstructured.Unstructured

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", oldpath, err)
	}

	expectedAPIVersion := "kubescheduler.config.k8s.io/v1"
	if apiVersion := config.GetAPIVersion(); apiVersion != expectedAPIVersion {
		return fmt.Errorf("invalid apiVersion in scheduler configuration at %s: expected %s, got %s", oldpath, expectedAPIVersion, apiVersion)
	}

	expectedKind := "KubeSchedulerConfiguration"
	if kind := config.GetKind(); kind != expectedKind {
		return fmt.Errorf("invalid kind in scheduler configuration at %s: expected %s, got %s", oldpath, expectedKind, kind)
	}

	clientConnection, ok := config.Object["clientConnection"]
	if !ok {
		clientConnection = map[string]any{}
	}

	clientConnectionMap, ok := clientConnection.(map[string]any)
	if !ok {
		clientConnectionMap = map[string]any{}
	}

	clientConnectionMap["kubeconfig"] = kubeconfig
	config.Object["clientConnection"] = clientConnectionMap
	updatedData, err := yaml.Marshal(config.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = c.WriteFileWithMode(newpath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", newpath, err)
	}

	return nil
}
