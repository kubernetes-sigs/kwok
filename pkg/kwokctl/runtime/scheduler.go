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

	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// CopySchedulerConfig copies the scheduler configuration file to the given path.
func (c *Cluster) CopySchedulerConfig(oldpath, newpath, kubeconfig string) error {
	data, err := c.ReadFile(oldpath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", oldpath, err)
	}

	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", oldpath, err)
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	if config["clientConnection"] == nil {
		config["clientConnection"] = make(map[string]interface{})
	}

	clientConnection, ok := config["clientConnection"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("clientConnection field is not a map in %s", oldpath)
	}

	// Only set kubeconfig if it doesn't already exist
	if _, exists := clientConnection["kubeconfig"]; !exists {
		clientConnection["kubeconfig"] = kubeconfig
	}

	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = c.WriteFileWithMode(newpath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", newpath, err)
	}

	return nil
}
