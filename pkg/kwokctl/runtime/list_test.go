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
		if err := os.MkdirAll(clusterPath, 0755); err != nil {
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
	defer os.RemoveAll(tmpDir)

	// Create some temporary cluster directories
	clusterDirs := []string{"cluster1", "cluster2"}
	for _, clusterDir := range clusterDirs {
		if err := os.MkdirAll(tmpDir+"/"+clusterDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Test GetUsedPorts function
	usedPorts := GetUsedPorts(ctx)
	if len(usedPorts) != 0 {
		t.Errorf("GetUsedPorts returned unexpected used ports: got %v, want empty set", usedPorts)
	}
}