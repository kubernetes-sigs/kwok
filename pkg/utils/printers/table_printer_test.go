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

package printers

import (
	"bytes"
	"io"
	"testing"
)

func TestWriteRows(t *testing.T) {
	buffer := &bytes.Buffer{}
	tablePrinter := NewTablePrinter(buffer)

	err := tablePrinter.WriteAll([][]string{
		{"Hello", "World"},
		{"Foo", "Bar"},
		{"Some long text", "Short"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "" +
		"Hello            World\n" +
		"Foo              Bar\n" +
		"Some long text   Short\n"

	if buffer.String() != want {
		t.Fatalf("unexpected output: %v", buffer.String())
	}
}

// TestWriteSingleRow tests writing a single row.
func TestWriteSingleRow(t *testing.T) {
	buffer := &bytes.Buffer{}
	tablePrinter := NewTablePrinter(buffer)

	err := tablePrinter.Write([]string{"Hello", "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Hello   World\n"
	if buffer.String() != want {
		t.Fatalf("unexpected output: got %q, want %q", buffer.String(), want)
	}
}

// TestWriteEmptyRow tests writing an empty row.
func TestWriteEmptyRow(t *testing.T) {
	buffer := &bytes.Buffer{}
	tablePrinter := NewTablePrinter(buffer)

	err := tablePrinter.Write([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "\n"
	if buffer.String() != want {
		t.Fatalf("unexpected output: got %q, want %q", buffer.String(), want)
	}
}
func TestWriteWithEmptyColumns(t *testing.T) {
	buffer := &bytes.Buffer{}
	tablePrinter := NewTablePrinter(buffer)

	err := tablePrinter.WriteAll([][]string{
		{"Hello", ""},
		{"", "World"},
		{"", ""},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "" +
		"Hello   \n" +
		"        World\n" +
		"        \n"

	if buffer.String() != want {
		t.Fatalf("unexpected output: got %q, want %q", buffer.String(), want)
	}
}

// TestWriteErrorPropagation tests if errors from the writer are propagated correctly.
func TestWriteErrorPropagation(t *testing.T) {
	errorWriter := &ErrorWriter{}
	tablePrinter := NewTablePrinter(errorWriter)

	err := tablePrinter.Write([]string{"Hello", "World"})
	if err == nil {
		t.Fatalf("expected an error but got nil")
	}
}

// ErrorWriter is a mock writer that always returns an error.
type ErrorWriter struct{}

func (e *ErrorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrShortWrite
}
