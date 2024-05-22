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

package file

import (
	"bytes"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	// Test creating a file
	testFileName := "testfile_create.txt"
	err := Create(testFileName)
	if err != nil {
		t.Errorf("Error creating file: %v", err)
	}
	defer func() {
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	// Check if the file exists
	if _, err := os.Stat(testFileName); os.IsNotExist(err) {
		t.Errorf("File %s was not created", testFileName)
	}
}

func TestCopy(t *testing.T) {
	// Create a test file
	srcFileName := "testfile_copy_src.txt"
	dstFileName := "testfile_copy_dst.txt"
	content := []byte("This is a test file for copying.")
	err := os.WriteFile(srcFileName, content, 0640)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer func() {
		if err := os.Remove(srcFileName); err != nil {
			t.Errorf("Error removing source file: %v", err)
		}
	}()
	defer func() {
		if err := os.Remove(dstFileName); err != nil {
			t.Errorf("Error removing destination file: %v", err)
		}
	}()

	// Copy the file
	err = Copy(srcFileName, dstFileName)
	if err != nil {
		t.Errorf("Error copying file: %v", err)
	}

	// Check if the content of the copied file is the same as the source file
	copiedContent, err := os.ReadFile(dstFileName)
	if err != nil {
		t.Errorf("Error reading copied file: %v", err)
	}

	if !bytes.Equal(copiedContent, content) {
		t.Errorf("Copied content does not match source content")
	}
}

func TestRename(t *testing.T) {
	// Create a test file
	oldFileName := "testfile_rename_old.txt"
	newFileName := "testfile_rename_new.txt"
	content := []byte("This is a test file for renaming.")
	err := os.WriteFile(oldFileName, content, 0640)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	defer func() {
		if _, err := os.Stat(oldFileName); err == nil {
			if err := os.Remove(oldFileName); err != nil {
				t.Errorf("Error removing old file: %v", err)
			}
		}
		if _, err := os.Stat(newFileName); err == nil {
			if err := os.Remove(newFileName); err != nil {
				t.Errorf("Error removing new file: %v", err)
			}
		}
	}()

	// Rename the file
	err = Rename(oldFileName, newFileName)
	if err != nil {
		t.Errorf("Error renaming file: %v", err)
	}

	// Check if the new file exists and old file does not exist
	if _, err := os.Stat(oldFileName); !os.IsNotExist(err) {
		t.Errorf("Old file %s still exists after renaming", oldFileName)
	}
	if _, err := os.Stat(newFileName); os.IsNotExist(err) {
		t.Errorf("New file %s does not exist after renaming", newFileName)
	}
}

func TestAppend(t *testing.T) {
	// Create a test file
	testFileName := "testfile_append.txt"
	initialContent := []byte("Initial content.\n")
	err := os.WriteFile(testFileName, initialContent, 0640)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	// Content to append
	appendContent := []byte("Additional content.\n")

	// Append content to the file
	err = Append(testFileName, appendContent)
	if err != nil {
		t.Errorf("Error appending content to file: %v", err)
	}

	// Read the file and check if the content is as expected
	fileContent, err := os.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expectedContent := append([]byte{}, initialContent...)
	expectedContent = append(expectedContent, appendContent...)
	if !bytes.Equal(fileContent, expectedContent) {
		t.Errorf("Appended content does not match expected content")
	}
}

func TestExists(t *testing.T) {
	// Test with an existing file
	existingFileName := "testfile_exists.txt"
	content := []byte("This is a test file for existence.")
	err := os.WriteFile(existingFileName, content, 0640)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(existingFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	exists := Exists(existingFileName)
	if !exists {
		t.Errorf("Exists() returned false for an existing file")
	}

	// Test with a non-existing file
	nonExistingFileName := "testfile_nonexistent.txt"
	exists = Exists(nonExistingFileName)
	if exists {
		t.Errorf("Exists() returned true for a non-existing file")
	}
}

func TestRemove(t *testing.T) {
	// Create a test file
	testFileName := "testfile_remove.txt"
	content := []byte("This is a test file for removal.")
	err := os.WriteFile(testFileName, content, 0640)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Ensure the file exists before attempting to remove it
	if _, err := os.Stat(testFileName); os.IsNotExist(err) {
		t.Fatalf("Test file %s does not exist", testFileName)
	}

	// Remove the file
	err = Remove(testFileName)
	if err != nil {
		t.Errorf("Error removing file: %v", err)
	}

	// Check if the file exists after removal
	if _, err := os.Stat(testFileName); !os.IsNotExist(err) {
		t.Errorf("File %s still exists after removal", testFileName)
	}
}

func TestOpen(t *testing.T) {
	// Test opening a file
	testFileName := "testfile_open.txt"
	file, err := Open(testFileName)
	if err != nil {
		t.Fatalf("Error opening file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("Error closing file: %v", err)
		}
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	// Write content to the file
	content := []byte("This is a test content.")
	_, err = file.Write(content)
	if err != nil {
		t.Errorf("Error writing to file: %v", err)
	}

	// Read the content and check if it matches
	fileContent, err := os.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	if !bytes.Equal(fileContent, content) {
		t.Errorf("Content read from file does not match expected content")
	}
}

func TestRead(t *testing.T) {
	// Create a test file
	testFileName := "testfile_read.txt"
	content := []byte("This is a test content for reading.")
	err := os.WriteFile(testFileName, content, 0640)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	// Read the content of the file
	fileContent, err := Read(testFileName)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}

	// Check if the content read matches the expected content
	if !bytes.Equal(fileContent, content) {
		t.Errorf("Content read from file does not match expected content")
	}
}

func TestWrite(t *testing.T) {
	// Create a test file
	testFileName := "testfile_write.txt"
	content := []byte("This is a test content for writing.")
	err := os.WriteFile(testFileName, []byte(""), 0640) // Create an empty file
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() {
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("Error removing test file: %v", err)
		}
	}()

	// Write content to the file
	err = Write(testFileName, content)
	if err != nil {
		t.Errorf("Error writing to file: %v", err)
	}

	// Read the content of the file and check if it matches
	fileContent, err := os.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}

	if !bytes.Equal(fileContent, content) {
		t.Errorf("Content written to file does not match expected content")
	}
}

func TestMkdirAll(t *testing.T) {
	// Test creating a directory
	testDirName := "testdir_mkdir_all"
	err := MkdirAll(testDirName)
	if err != nil {
		t.Fatalf("Error creating directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(testDirName); err != nil {
			t.Errorf("Error removing test directory: %v", err)
		}
	}()

	// Check if the directory exists
	exists := Exists(testDirName)
	if !exists {
		t.Errorf("Directory %s was not created", testDirName)
	}
}
