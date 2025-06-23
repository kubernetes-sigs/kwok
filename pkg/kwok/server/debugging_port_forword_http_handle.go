/*
Copyright 2025 The Kubernetes Authors.

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

package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

func httpPortForwardHTTPHandle(ctx context.Context, handler http.Handler, stream io.ReadWriteCloser) error {
	server := newHttpServeConn(&http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           handler,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	})

	conn, ok := stream.(net.Conn)
	if !ok {
		return fmt.Errorf("stream %T is not a net.Conn", conn)
	}

	server.ServeConn(conn)
	return nil
}

type singleConnListener struct {
	addr net.Addr
	ch   chan net.Conn
	once sync.Once
}

func newSingleConnListener(conn net.Conn) net.Listener {
	ch := make(chan net.Conn, 1)
	ch <- conn
	return &singleConnListener{
		addr: conn.LocalAddr(),
		ch:   ch,
	}
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	conn, ok := <-l.ch
	if !ok || conn == nil {
		return nil, errors.New("use of closed network connection")
	}
	return &connCloser{
		l:    l,
		Conn: conn,
	}, nil
}

func (l *singleConnListener) shutdown() {
	l.once.Do(func() {
		close(l.ch)
	})
}

func (l *singleConnListener) Close() error {
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.addr
}

type connCloser struct {
	l *singleConnListener
	net.Conn
}

func (c *connCloser) Close() error {
	c.l.shutdown()
	return c.Conn.Close()
}

func (c *connCloser) SetDeadline(t time.Time) error {
	return nil
}

func (c *connCloser) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *connCloser) SetWriteDeadline(t time.Time) error {
	return nil
}

type httpServeConn struct {
	*http.Server
}

func newHttpServeConn(s *http.Server) *httpServeConn {
	return &httpServeConn{s}
}

func (w httpServeConn) ServeConn(conn net.Conn) {
	listener := newSingleConnListener(conn)
	_ = w.Serve(listener)
}
