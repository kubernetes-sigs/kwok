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
	"path/filepath"
	"testing"
)

func TestCopySchedulerConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Define paths for old and new scheduler configuration files
	oldSchedulerConfig := filepath.Join(tmpDir, "old_scheduler_config.yaml")
	newSchedulerConfig := filepath.Join(tmpDir, "new_scheduler_config.yaml")
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")

	// Create a temporary old scheduler configuration file
	err := os.WriteFile(oldSchedulerConfig, []byte("old scheduler config"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a temporary kubeconfig file
	err = os.WriteFile(kubeconfigPath, []byte("kubeconfig"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test CopySchedulerConfig function
	err = cluster.CopySchedulerConfig(oldSchedulerConfig, newSchedulerConfig, kubeconfigPath)
	if err != nil {
		t.Fatalf("CopySchedulerConfig returned an unexpected error: %v", err)
	}

	// Verify that the new scheduler configuration file exists
	_, err = os.Stat(newSchedulerConfig)
	if err != nil {
		t.Fatalf("New scheduler configuration file does not exist: %v", err)
	}

	// Read the contents of the new scheduler configuration file
	newConfigBytes, err := os.ReadFile(newSchedulerConfig)
	if err != nil {
		t.Fatalf("Failed to read new scheduler configuration file: %v", err)
	}

	// Define the expected content of the new scheduler configuration file
	expectedConfig := fmt.Sprintf("old scheduler config\nclientConnection:\n  kubeconfig: %q\n", kubeconfigPath)

	// Compare the actual content with the expected content
	if string(newConfigBytes) != expectedConfig {
		t.Errorf("Unexpected content in the new scheduler configuration file: got %q, want %q", string(newConfigBytes), expectedConfig)
	}
}
