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

package dryrun

import (
	"bytes"
	"testing"
)

// TestIsCatToFileWriter tests the IsCatToFileWriter function
func TestIsCatToFileWriter(t *testing.T) {
	writer := newCatToFileWriter(&bytes.Buffer{}, "testfile")

	name, ok := IsCatToFileWriter(writer)
	if !ok || name != "testfile" {
		t.Errorf("expected true and 'testfile', got %v and %q", ok, name)
	}

	_, ok = IsCatToFileWriter(&bytes.Buffer{})
	if ok {
		t.Errorf("expected false, got %v", ok)
	}
}

// TestDryRunWriterWriteAndClose tests the Write and Close methods of dryRunWriter
func TestDryRunWriterWriteAndClose(t *testing.T) {
	var buf bytes.Buffer
	writer := newCatToFileWriter(&buf, "testfile")

	data := []byte("test data")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "echo test data >testfile\n"
	actual := buf.String()
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

// TestDryRunWriterCloseWithNewline tests the Close method of dryRunWriter with newline in data
func TestDryRunWriterCloseWithNewline(t *testing.T) {
	var buf bytes.Buffer
	writer := newCatToFileWriter(&buf, "testfile")

	data := []byte("line1\nline2")
	_, _ = writer.Write(data)

	err := writer.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "cat <<EOF >testfile\nline1\nline2\nEOF\n"
	actual := buf.String()
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

// TestNewCatToFileWriter tests the NewCatToFileWriter function
func TestNewCatToFileWriter(t *testing.T) {
	writer := NewCatToFileWriter("testfile")
	defer func() {
		if err := writer.Close(); err != nil {
			t.Fatalf("Failed to close : %v", err)
		}
	}()

	if _, ok := writer.(*dryRunWriter); !ok {
		t.Errorf("expected *dryRunWriter, got %T", writer)
	}
}
