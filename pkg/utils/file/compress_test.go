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
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		data     []byte
		expected bool
	}{{
		name:     "test.txt",
		ext:      ".txt",
		data:     []byte("hello world"),
		expected: false,
	},
		{
			name:     "test.gz",
			ext:      ".gz",
			data:     []byte("hello world"),
			expected: true,
		},
		{
			name:     "test.tgz",
			ext:      ".tgz",
			data:     []byte("hello world"),
			expected: true,
		},
		{
			name:     "test.zip",
			ext:      ".zip",
			data:     []byte("hello world"),
			expected: false,
		},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		writer := Compress(test.name, &buf)
		_, err := writer.Write(test.data)
		if err != nil {
			t.Fatalf("failed to write data: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("failed to close: %v", err)
		}

		if test.expected {
			gr, err := gzip.NewReader(&buf)
			if err != nil {
				t.Fatalf("expected gzip compression, got error: %v", err)
			}
			defer func() {
				if err := gr.Close(); err != nil {
					t.Fatalf("Failed to close : %v", err)
				}
			}()
			data, err := io.ReadAll(gr)
			if err != nil {
				t.Fatalf("failed to read gzip data: %v", err)
			}
			if string(data) != "hello world" {
				t.Errorf("expected 'hello world', got %s", string(data))
			}
		} else if buf.String() != "hello world" {
			t.Errorf("expected 'hello world', got %s", buf.String())
		}
	}
}

func TestDecompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte

		compressed bool
	}{{

		name:       "test.txt",
		data:       []byte("hello world"),
		compressed: false,
	},
		{

			name:       "test.gz",
			data:       []byte("hello world"),
			compressed: true,
		},
		{

			name:       "test.tgz",
			data:       []byte("hello world"),
			compressed: true,
		},

		{

			name:       "test.zip",
			data:       []byte("hello world"),
			compressed: false,
		},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		if test.compressed {
			gw := gzip.NewWriter(&buf)
			_, err := gw.Write(test.data)
			if err != nil {
				t.Fatalf("failed to write gzip data: %v", err)
			}

			if err := gw.Close(); err != nil {
				t.Fatalf("failed to close: %v", err)
			}
		} else {
			buf.WriteString("hello world")
		}

		reader, err := Decompress(test.name, &buf)
		if err != nil {
			t.Fatalf("failed to decompress data: %v", err)
		}

		defer func() {
			if err := reader.Close(); err != nil {
				t.Fatalf("Failed to close : %v", err)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("failed to read decompressed data: %v", err)
		}
		if string(data) != "hello world" {
			t.Errorf("expected 'hello world', got %s", string(data))
		}
	}
}
