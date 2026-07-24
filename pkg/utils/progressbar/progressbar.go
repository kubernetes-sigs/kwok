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
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// NewTransport returns a new transport that writes a progress bar to out.
func NewTransport(base http.RoundTripper) http.RoundTripper {
	return NewTransportWithOffset(base, 0)
}

// NewTransportWithOffset returns a new transport that writes a progress bar starting at offset.
func NewTransportWithOffset(base http.RoundTripper, offset uint64) http.RoundTripper {
	out := os.Stderr
	if !term.IsTerminal(int(out.Fd())) {
		return base
	}
	return &transport{
		base:   base,
		offset: offset,
	}
}

type transport struct {
	base   http.RoundTripper
	offset uint64
}

func (p *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := p.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, nil
	}

	if strings.HasSuffix(req.URL.Path, "/") {
		return resp, nil
	}

	contentLength, _ := strconv.ParseUint(resp.Header.Get("Content-Length"), 10, 64)
	if contentLength == 0 {
		return resp, nil
	}

	var name string
	ref := req.Referer()
	if ref != "" {
		name = path.Base(ref)
	} else {
		name = path.Base(req.URL.Path)
	}

	offset := uint64(0)
	if resp.StatusCode == http.StatusPartialContent {
		offset = p.offset
	}
	resp.Body = NewReadCloserWithOffset(resp.Body, name, contentLength+offset, offset)

	return resp, nil
}
