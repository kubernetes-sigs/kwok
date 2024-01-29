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

package recording

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	formatRFC3339Nano  = time.RFC3339Nano    // "2006-01-02T15:04:05.999999999Z07:00"
	formatRFC3339Micro = metav1.RFC3339Micro //  "2006-01-02T15:04:05.000000Z07:00"

	regReplaceTimeFormat = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`)
	regRevertTimeOffset  = regexp.MustCompile(`\$\(time-offset-nanosecond -?\d+\)`)
)

// ReplaceTimeToRelative replaces the time to relative.
func ReplaceTimeToRelative(baseTime time.Time, data []byte) []byte {
	return regReplaceTimeFormat.ReplaceAllFunc(data, func(s []byte) []byte {
		t, err := time.Parse(formatRFC3339Nano, string(s))
		if err != nil {
			return s
		}

		sub := t.Sub(baseTime)
		return []byte(fmt.Sprintf("$(time-offset-nanosecond %d)", int64(sub)))
	})
}

// RevertTimeFromRelative reverts the time from relative to absolute.
func RevertTimeFromRelative(baseTime time.Time, data []byte) []byte {
	return regRevertTimeOffset.ReplaceAllFunc(data, func(s []byte) []byte {
		// $(time-offset-nanosecond 0)
		i, err := strconv.ParseInt(string(s[25:len(s)-1]), 0, 0)
		if err != nil {
			return s
		}

		t := baseTime.Add(time.Duration(i)).UTC()
		return []byte(t.Format(formatRFC3339Micro))
	})
}

// NewWriteHook creates a new write hook.
func NewWriteHook(w io.Writer, hook func([]byte) []byte) io.Writer {
	return &writeHook{
		w:    w,
		hook: hook,
	}
}

type writeHook struct {
	hook func([]byte) []byte
	w    io.Writer
}

func (w *writeHook) Write(data []byte) (int, error) {
	d := w.hook(data)
	n, err := w.w.Write(d)
	if err != nil {
		return n, err
	}
	if n != len(d) {
		return n, io.ErrShortWrite
	}
	return len(data), nil
}

type readHook struct {
	hook func([]byte) []byte
	r    *bufio.Reader
	buf  bytes.Buffer
}

// NewReadHook creates a new read hook.
func NewReadHook(r io.Reader, hook func([]byte) []byte) io.Reader {
	return &readHook{
		r:    bufio.NewReader(r),
		hook: hook,
	}
}

func (r *readHook) Read(data []byte) (int, error) {
	if r.buf.Len() != 0 {
		return r.buf.Read(data)
	}
	line, err := r.r.ReadSlice('\n')
	if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
		return 0, err
	}
	d := r.hook(line)
	r.buf.Write(d)
	return r.buf.Read(data)
}
