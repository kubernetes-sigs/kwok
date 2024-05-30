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
	"os"
	"path/filepath"
	"testing"
)

// TestCreate tests the Create function.
func TestCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")
	err = Create(filePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to be created, but it does not exist")
	}
}

// TestCopy tests the Copy function.
func TestCopy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	srcFilePath := filepath.Join(tmpDir, "srcfile.txt")
	dstFilePath := filepath.Join(tmpDir, "dstfile.txt")

	content := []byte("Hello, World!")
	err = os.WriteFile(srcFilePath, content, 0640)
	if err != nil {
		t.Fatal(err)
	}

	err = Copy(srcFilePath, dstFilePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	dstContent, err := os.ReadFile(dstFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(dstContent) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, dstContent)
	}
}

// TestRename tests the Rename function.
func TestRename(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	oldFilePath := filepath.Join(tmpDir, "oldfile.txt")
	newFilePath := filepath.Join(tmpDir, "newfile.txt")

	content := []byte("Hello, World!")
	err = os.WriteFile(oldFilePath, content, 0640)
	if err != nil {
		t.Fatal(err)
	}

	err = Rename(oldFilePath, newFilePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
		t.Fatalf("Expected file to be renamed, but it does not exist")
	}
}

// TestAppend tests the Append function.
func TestAppend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	content1 := []byte("Hello")
	content2 := []byte(", World!")
	expectedContent := []byte("Hello, World!")

	err = os.WriteFile(filePath, content1, 0640)
	if err != nil {
		t.Fatal(err)
	}

	err = Append(filePath, content2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	finalContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(finalContent) != string(expectedContent) {
		t.Fatalf("Expected content %q, got %q", expectedContent, finalContent)
	}
}

// TestExists tests the Exists function.
func TestExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	if Exists(filePath) {
		t.Fatalf("Expected file to not exist")
	}

	err = os.WriteFile(filePath, []byte("content"), 0640)
	if err != nil {
		t.Fatal(err)
	}

	if !Exists(filePath) {
		t.Fatalf("Expected file to exist")
	}
}

// TestRemove tests the Remove function.
func TestRemove(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	err = os.WriteFile(filePath, []byte("content"), 0640)
	if err != nil {
		t.Fatal(err)
	}

	err = Remove(filePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if Exists(filePath) {
		t.Fatalf("Expected file to be removed")
	}
}

// TestRemoveAll tests the RemoveAll function.
func TestRemoveAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	dirPath := filepath.Join(tmpDir, "testdir")
	err = MkdirAll(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	if !Exists(dirPath) {
		t.Fatalf("Expected directory to exist")
	}

	err = RemoveAll(dirPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if Exists(dirPath) {
		t.Fatalf("Expected directory to be removed")
	}
}

// TestOpen tests the Open function.
func TestOpen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	file, err := Open(filePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	content := []byte("Hello, World!")
	_, err = file.Write(content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(readContent) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, readContent)
	}
}

// TestRead tests the Read function.
func TestRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	content := []byte("Hello, World!")
	err = os.WriteFile(filePath, content, 0640)
	if err != nil {
		t.Fatal(err)
	}

	readContent, err := Read(filePath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(readContent) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, readContent)
	}
}

// TestWrite tests the Write function.
func TestWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	content := []byte("Hello, World!")
	err = Write(filePath, content)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(readContent) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, readContent)
	}
}

// TestWriteWithMode tests the WriteWithMode function.
func TestWriteWithMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	filePath := filepath.Join(tmpDir, "testfile.txt")

	content := []byte("Hello, World!")
	mode := os.FileMode(0644)
	err = WriteWithMode(filePath, content, mode)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode() != mode {
		t.Fatalf("Expected mode %v, got %v", mode, info.Mode())
	}

	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(readContent) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, readContent)
	}
}

// TestMkdirAll tests the MkdirAll function.
func TestMkdirAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	dirPath := filepath.Join(tmpDir, "parent", "child")

	err = MkdirAll(dirPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !Exists(dirPath) {
		t.Fatalf("Expected directory to be created")
	}
}
