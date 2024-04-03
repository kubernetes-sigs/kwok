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
	"time"

	"golang.org/x/term"
)

type reader struct {
	reader  io.Reader
	current uint64

	name  string
	total uint64

	startTime      time.Time
	lastUpdateTime time.Time
	out            *os.File
}

// NewReader returns a new reader that writes a progress bar to out.
func NewReader(r io.Reader, name string, total uint64) io.Reader {
	out := os.Stderr
	if !term.IsTerminal(int(out.Fd())) {
		return r
	}

	return &reader{
		reader:    r,
		name:      name,
		total:     total,
		startTime: time.Now(),
		out:       out,
	}
}

func (r *reader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	if n == 0 {
		return n, err
	}
	r.current += uint64(n)

	if r.current != r.total && time.Since(r.lastUpdateTime) < time.Second*10 {
		return n, err
	}

	termWidth, _, _ := term.GetSize(int(r.out.Fd()))
	if termWidth > 0 {
		info := formatProgress(r.name, uint64(termWidth), r.current, r.total, time.Since(r.startTime))
		if r.current == r.total {
			_, _ = r.out.WriteString("\r" + info + "\n")
		} else {
			_, _ = r.out.WriteString("\r" + info)
		}
	}
	return n, err
}

// NewReadCloser returns a new ReadCloser that writes a progress bar to out.
func NewReadCloser(rc io.ReadCloser, name string, total uint64) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: NewReader(rc, name, total),
		Closer: rc,
	}
}
