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

package dryrun

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

var stdout = os.Stdout

// DryRun is a flag to indicate whether the program is running in dry-run mode.
var DryRun bool

// PrintMessage prints the message to stdout.
func PrintMessage(format string, a ...any) {
	_, _ = fmt.Fprintf(stdout, format+"\n", a...)
}

type dryRunWriter struct {
	name string
	w    io.Writer
	buf  bytes.Buffer
}

// IsCatToFileWriter returns true if the writer is a cat to file writer.
func IsCatToFileWriter(w io.Writer) (string, bool) {
	d, ok := w.(*dryRunWriter)
	if !ok {
		return "", false
	}
	return d.name, true
}

func newCatToFileWriter(w io.Writer, name string) *dryRunWriter {
	return &dryRunWriter{
		name: name,
		w:    w,
	}
}

func (d *dryRunWriter) Write(p []byte) (n int, err error) {
	return d.buf.Write(p)
}

func (d *dryRunWriter) Close() error {
	buf := d.buf.String()
	if len(buf) == 0 {
		return nil
	}

	line := strings.TrimSpace(buf)
	if !strings.Contains(line, "\n") {
		_, _ = fmt.Fprintf(d.w, "echo %s >%s\n", line, d.name)
		return nil
	}
	_, _ = fmt.Fprintf(d.w, "cat <<EOF >%s\n%s\nEOF\n", d.name, buf)
	return nil
}

// NewCatToFileWriter returns a writer that prints the content to stdout.
func NewCatToFileWriter(name string) io.WriteCloser {
	return newCatToFileWriter(stdout, name)
}
