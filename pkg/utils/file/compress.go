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
	"bufio"
	"compress/gzip"
	"io"
	"path/filepath"
)

// Compress compresses a writer
func Compress(name string, w io.Writer) io.WriteCloser {
	ext := filepath.Ext(name)
	switch ext {
	case ".gz", ".tgz":
		z, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
		return z
	}

	return struct {
		io.Writer
		io.Closer
	}{
		Writer: w,
		Closer: nopCloser{},
	}
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

const (
	gzipID1 = 0x1f
	gzipID2 = 0x8b
)

// Decompress decompresses a reader
// It will return a raw reader if the reader is not compressed
func Decompress(name string, r io.Reader) (io.ReadCloser, error) {
	ext := filepath.Ext(name)
	switch ext {
	case ".gz", ".tgz":
		bufReader := bufio.NewReader(r)

		prefix, err := bufReader.Peek(2)
		if err != nil {
			return nil, err
		}

		if prefix[0] == gzipID1 && prefix[1] == gzipID2 {
			return gzip.NewReader(bufReader)
		}

		return io.NopCloser(bufReader), nil
	}

	return io.NopCloser(r), nil
}
