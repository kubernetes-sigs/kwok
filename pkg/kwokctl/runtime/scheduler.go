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

	v1 "k8s.io/kube-scheduler/config/v1"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// CopySchedulerConfig copies the scheduler configuration file to the given path.
func (c *Cluster) CopySchedulerConfig(oldpath, newpath, kubeconfig string) error {
	var config v1.KubeSchedulerConfiguration

	data, err := os.ReadFile(oldpath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", oldpath, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", oldpath, err)
	}

	expectedAPIVersion := v1.SchemeGroupVersion.String()
	if config.APIVersion != expectedAPIVersion {
		return fmt.Errorf("invalid apiVersion in scheduler configuration at %s: expected %s, got %s", oldpath, expectedAPIVersion, config.APIVersion)
	}

	expectedKind := "KubeSchedulerConfiguration"
	if config.Kind != expectedKind {
		return fmt.Errorf("invalid kind in scheduler configuration at %s: expected %s, got %s", oldpath, expectedKind, config.Kind)
	}

	if config.ClientConnection.Kubeconfig != "" {
		return fmt.Errorf("kubeconfig already exists in scheduler configuration at %s", oldpath)
	}

	config.ClientConnection.Kubeconfig = kubeconfig
	updatedData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = c.WriteFile(newpath, updatedData)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", newpath, err)
	}

	return nil
}
