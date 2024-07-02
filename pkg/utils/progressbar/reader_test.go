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
package progressbar

import (
	"io"
	"os"
	"testing"
	"time"
)

// TestReadProgress verifies that the Read function updates the progress bar appropriately.
func TestReadProgress(t *testing.T) {
	// Mock reader, name, and total size
	mockReader := &mockReader{}
	name := "testFile"
	total := uint64(100)

	// Create new reader with progress bar
	reader := &reader{
		reader:    mockReader,
		name:      name,
		total:     total,
		startTime: time.Now(),
		out:       os.Stderr,
	}

	// Simulate reading some data
	data := make([]byte, 20) // Read 20 bytes
	n, err := reader.Read(data)
	if err != nil {
		t.Errorf("Error while reading data: %v", err)
	}

	// Validate that the progress bar was updated
	expectedCurrent := uint64(n)
	if reader.current != expectedCurrent {
		t.Errorf("Progress bar was not updated correctly. Expected: %d, Got: %d", expectedCurrent, reader.current)
	}
}

// TestNewReader verifies the NewReader function.
func TestNewReader(t *testing.T) {
	// Mock reader and progress bar parameters
	r := &mockReader{}
	name := "testFile"
	total := uint64(100)

	// Create new reader with progress bar
	reader := NewReader(r, name, total)

	// As reader is already of type io.Reader, no need for type assertion
	// We can directly use it as an io.Reader
	_, err := reader.Read(make([]byte, 0)) // Perform a read operation to ensure reader is functional
	if err != nil {
		t.Errorf("NewReader did not return a functional reader: %v", err)
	}
}

// TestNewReadCloser verifies the NewReadCloser function.
func TestNewReadCloser(t *testing.T) {
	// Mock ReadCloser, name, and total size
	rc := &mockReadCloser{}
	name := "testFile"
	total := uint64(100)

	// Create new ReadCloser with progress bar
	readCloser := NewReadCloser(rc, name, total)

	// Verify that ReadCloser implements both io.Reader and io.Closer interfaces
	_, okReader := readCloser.(io.Reader)
	if !okReader {
		t.Error("NewReadCloser did not return an io.Reader")
	}
	_, okCloser := readCloser.(io.Closer)
	if !okCloser {
		t.Error("NewReadCloser did not return an io.Closer")
	}
}

// Mock Reader for testing
type mockReader struct{}

func (m *mockReader) Read(b []byte) (int, error) {
	return 0, nil
}

// Mock ReadCloser for testing
type mockReadCloser struct{}

func (m *mockReadCloser) Read(b []byte) (int, error) {
	return 0, nil
}

func (m *mockReadCloser) Close() error {
	return nil
}
