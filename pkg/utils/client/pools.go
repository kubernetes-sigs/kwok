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

package client

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/pools"
)

type roundTripperPool struct {
	p     *pools.Pool[http.RoundTripper]
	count atomic.Int32
}

func newRoundTripperPool(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &roundTripperPool{
		p: pools.NewPool(func() http.RoundTripper {
			return cloneRoundTripper(rt)
		}),
	}
}

func (p *roundTripperPool) RoundTrip(req *http.Request) (*http.Response, error) {
	logger := log.FromContext(req.Context())
	start := time.Now()
	p.count.Add(1)

	t := p.p.Get()

	resp, err := t.RoundTrip(req)
	if err != nil {
		logger.Info("Hello", "URL", req.URL, "elapsed", time.Since(start), "pool", p.count.Add(-1))
		p.p.Put(t)
		return nil, err
	}

	if resp.Body == nil {
		logger.Info("Hello", "URL", req.URL, "elapsed", time.Since(start), "pool", p.count.Add(-1))
		p.p.Put(t)
		return resp, nil
	}

	resp.Body = &responseBody{
		fun: func() {
			logger.Info("Hello", "URL", req.URL, "elapsed", time.Since(start), "pool", p.count.Add(-1))
			p.p.Put(t)
		},
		rc: resp.Body,
	}
	return resp, nil
}

func cloneRoundTripper(rt http.RoundTripper) http.RoundTripper {
	transport, isTransport := rt.(*http.Transport)
	if !isTransport {
		panic(fmt.Sprintf("unexpected non-http transport %T", rt))
	}

	var dial net.Dialer
	t := transport.Clone()
	t.DialContext = dial.DialContext

	return t
}

type responseBody struct {
	o   sync.Once
	fun func()
	rc  io.ReadCloser
	err error
}

func (b *responseBody) cleanup() {
	b.o.Do(func() {
		b.err = b.rc.Close()
		b.fun()
	})
}

func (b *responseBody) Read(p []byte) (n int, err error) {
	n, err = b.rc.Read(p)
	if err != nil {
		b.cleanup()
	}
	return n, err
}

func (b *responseBody) Close() error {
	b.cleanup()
	return b.err
}
