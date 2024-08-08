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
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
)

func TestListClusters(t *testing.T) {
	ctx := context.TODO()

	// Create a temporary directory for testing clusters
	tmpDir := t.TempDir()

	// Set the clusters directory to the temporary directory
	config.ClustersDir = tmpDir

	// Create some temporary cluster directories and config files
	clusterDirs := []string{"cluster1", "cluster2"}
	for _, clusterDir := range clusterDirs {
		clusterPath := filepath.Join(tmpDir, clusterDir)
		if err := os.MkdirAll(clusterPath, 0750); err != nil {
			t.Fatal(err)
		}

		// Create a config file in each cluster directory
		configFilePath := filepath.Join(clusterPath, consts.ConfigName)
		if _, err := os.Create(configFilePath); err != nil {
			t.Fatal(err)
		}
	}

	// Test ListClusters function
	clusters, err := ListClusters(ctx)
	if err != nil {
		t.Fatalf("ListClusters returned an unexpected error: %v", err)
	}

	// Define the expected list of clusters
	expectedClusters := []string{"cluster1", "cluster2"}

	// Compare the actual clusters with the expected clusters
	if !reflect.DeepEqual(clusters, expectedClusters) {
		t.Errorf("ListClusters returned unexpected clusters: got %v, want %v", clusters, expectedClusters)
	}
}

func TestGetUsedPorts(t *testing.T) {
	ctx := context.TODO()

	// Create a temporary directory for testing clusters
	tmpDir := t.TempDir()
	// Defer the removal of the temporary directory
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("error removing temporary directory: %v", err)
		}
	}()

	// Create some temporary cluster directories
	clusterDirs := []string{"cluster1", "cluster2"}
	for _, clusterDir := range clusterDirs {
		if err := os.MkdirAll(tmpDir+"/"+clusterDir, 0750); err != nil {
			t.Fatal(err)
		}
	}

	// Test GetUsedPorts function
	usedPorts := GetUsedPorts(ctx)
	if len(usedPorts) != 0 {
		t.Errorf("GetUsedPorts returned unexpected used ports: got %v, want empty set", usedPorts)
	}
}
