/*
Copyright 2024 The Kubernetes Authors.

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
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUntar(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		files    []testFile // list of test files to create
		expected []testFile // list of expected files after extraction
		wantErr  bool
	}{
		{
			name: "untar .tar.gz file",
			src:  "testdata/test.tar.gz",
			files: []testFile{
				{"file1.txt", "content1"},
				{"file2.txt", "content2"},
			},
			expected: []testFile{
				{"output/file1.txt", "content1"},
				{"output/file2.txt", "content2"},
			},
			wantErr: false,
		},
		{
			name:     "unsupported file format",
			src:      "testdata/test.unsupported",
			files:    nil,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare test environment
			ctx := context.Background()
			if tt.files != nil {
				createTarGz(t, tt.src, tt.files)
			}
			defer os.Remove(tt.src)

			err := untar(ctx, tt.src, func(file string) (string, bool) {
				return "output/" + file, true
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("untar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify extracted files
			for _, expected := range tt.expected {
				content, err := os.ReadFile(expected.name)
				if err != nil {
					t.Fatalf("failed to read file %s: %v", expected.name, err)
				}
				if string(content) != expected.content {
					t.Errorf("expected content %s, got %s", expected.content, string(content))
				}
			}
		})
	}
}

// testFile represents a file with its name and content for testing purposes.
type testFile struct {
	name    string
	content string
}

// createTarGz creates a .tar.gz file with specified test files and their content.
func createTarGz(t *testing.T, src string, files []testFile) {
	// Create the directory structure
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create the .tar.gz file
	tarGzFile, err := os.Create(src)
	if err != nil {
		t.Fatalf("failed to create .tar.gz file: %v", err)
	}
	defer tarGzFile.Close()

	// Create the gzip writer
	gzw := gzip.NewWriter(tarGzFile)
	defer gzw.Close()

	// Create the tar writer
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// Write each test file to the tar archive
	for _, file := range files {
		header := &tar.Header{
			Name: file.name,
			Mode: 0644,
			Size: int64(len(file.content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write header for %s: %v", file.name, err)
		}
		if _, err := tw.Write([]byte(file.content)); err != nil {
			t.Fatalf("failed to write content for %s: %v", file.name, err)
		}
	}
}
