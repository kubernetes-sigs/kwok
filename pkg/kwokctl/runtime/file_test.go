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
	"os"
	"path/filepath"
	"testing"
)

func TestCreateFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Define the path for the test file
	testFilePath := filepath.Join(tmpDir, "test.txt")

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test CreateFile function
	err := cluster.CreateFile(testFilePath)
	if err != nil {
		t.Fatalf("CreateFile returned an unexpected error: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Errorf("CreateFile did not create the file as expected")
	}
}

func TestCopyFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Define the path for the test files
	srcFilePath := filepath.Join(tmpDir, "src.txt")
	destFilePath := filepath.Join(tmpDir, "dest.txt")

	// Create a mock source file
	if err := os.WriteFile(srcFilePath, []byte("source"), 0640); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test CopyFile function
	err := cluster.CopyFile(srcFilePath, destFilePath)
	if err != nil {
		t.Fatalf("CopyFile returned an unexpected error: %v", err)
	}

	// Check if the destination file exists and has the same content as the source file
	destContent, err := os.ReadFile(destFilePath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	srcContent, err := os.ReadFile(srcFilePath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	if string(destContent) != string(srcContent) {
		t.Errorf("CopyFile did not copy the file content as expected")
	}
}

func TestRenameFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Define the path for the test files
	oldFilePath := filepath.Join(tmpDir, "old.txt")
	newFilePath := filepath.Join(tmpDir, "new.txt")

	// Create a mock source file
	if err := os.WriteFile(oldFilePath, []byte("content"), 0640); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test RenameFile function
	err := cluster.RenameFile(oldFilePath, newFilePath)
	if err != nil {
		t.Fatalf("RenameFile returned an unexpected error: %v", err)
	}

	// Check if the old file exists and the new file exists after renaming
	if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
		t.Errorf("RenameFile did not remove the old file as expected")
	}
	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Errorf("RenameFile did not create the new file as expected")
	}
}

func TestAppendToFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Define the path for the test file
	testFilePath := filepath.Join(tmpDir, "test.txt")

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Write initial content to the file
	initialContent := []byte("initial content")
	if err := os.WriteFile(testFilePath, initialContent, 0640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Define content to append
	appendContent := []byte("appended content")

	// Test AppendToFile function
	err := cluster.AppendToFile(testFilePath, appendContent)
	if err != nil {
		t.Fatalf("AppendToFile returned an unexpected error: %v", err)
	}

	// Read the file to verify the appended content
	fileContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Check if the content contains both initial and appended content
	compareFileContent(t, initialContent, appendContent, fileContent)
}

func compareFileContent(t *testing.T, initialContent, appendContent, fileContent []byte) {
	expectedContent := append(initialContent, appendContent...)
	if string(fileContent) != string(expectedContent) {
		t.Errorf("AppendToFile did not append the content to the file as expected")
	}
}

func TestRemove(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Define the path for the test file
	testFilePath := filepath.Join(tmpDir, "test.txt")

	// Create a mock file
	if _, err := os.Create(testFilePath); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test Remove function
	err := cluster.Remove(testFilePath)
	if err != nil {
		t.Fatalf("Remove returned an unexpected error: %v", err)
	}

	// Check if the file exists after removal
	if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
		t.Errorf("Remove did not remove the file as expected")
	}
}

func TestRemoveAll(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a mock subdirectory and file
	subDir := filepath.Join(tmpDir, "subdir")
	testFilePath := filepath.Join(subDir, "test.txt")

	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if _, err := os.Create(testFilePath); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a mock Cluster instance
	cluster := &Cluster{}

	// Test RemoveAll function
	err := cluster.RemoveAll(tmpDir)
	if err != nil {
		t.Fatalf("RemoveAll returned an unexpected error: %v", err)
	}

	// Check if the directory and its contents exist after removal
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Errorf("RemoveAll did not remove the directory as expected")
	}
}
