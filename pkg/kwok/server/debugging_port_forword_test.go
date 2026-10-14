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

package server

import (
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
)

func Test_findPortInForwards(t *testing.T) {
	type args struct {
		port     int32
		forwards []internalversion.Forward
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.Forward
		wantOk bool
	}{
		{
			name: "find port in forwards",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{8080},
					},
				},
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
			wantOk: true,
		},
		{
			name: "not find port in forwards",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{8081},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find port in empty forwards",
			args: args{
				port:     8080,
				forwards: []internalversion.Forward{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find port in empty ports",
			args: args{
				port: 8080,
				forwards: []internalversion.Forward{
					{
						Ports: []int32{},
					},
				},
			},
			want: &internalversion.Forward{
				Ports: []int32{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findPortInForwards(tt.args.port, tt.args.forwards)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findPortInForwards() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findPortInForwards() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getPodsForward(t *testing.T) {
	type args struct {
		rules        []*internalversion.PortForward
		clusterRules []*internalversion.ClusterPortForward
		podName      string
		podNamespace string
		port         int32
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.Forward
		wantErr bool
	}{
		{
			name: "find port in rules",
			args: args{
				rules: []*internalversion.PortForward{
					{
						Name:      "test",
						Namespace: "default",
						Spec: internalversion.PortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				clusterRules: []*internalversion.ClusterPortForward{},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
		},
		{
			name: "find port in cluster rules",
			args: args{
				rules: []*internalversion.PortForward{},
				clusterRules: []*internalversion.ClusterPortForward{
					{
						Name: "cluster-test",
						Spec: internalversion.ClusterPortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want: &internalversion.Forward{
				Ports: []int32{8080},
			},
		},
		{
			name: "not find port in rules",
			args: args{
				rules: []*internalversion.PortForward{
					{
						Name:      "test-2",
						Namespace: "default",
						Spec: internalversion.PortForwardSpec{
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				clusterRules: []*internalversion.ClusterPortForward{},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find port in cluster rules",
			args: args{
				rules: []*internalversion.PortForward{},
				clusterRules: []*internalversion.ClusterPortForward{
					{
						Name: "cluster-test",
						Spec: internalversion.ClusterPortForwardSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Forwards: []internalversion.Forward{
								{
									Ports: []int32{8080},
								},
							},
						},
					},
				},
				podName:      "test",
				podNamespace: "default",
				port:         8080,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPodsForward(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodsForward() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodsForward() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// failingPortForwardStream keeps client input open after its output fails.
type failingPortForwardStream struct {
	reader *io.PipeReader
	err    error
	closed chan struct{}
	once   sync.Once
}

func (s *failingPortForwardStream) Read(buf []byte) (int, error) {
	return s.reader.Read(buf)
}

func (s *failingPortForwardStream) Write([]byte) (int, error) {
	return 0, s.err
}

func (s *failingPortForwardStream) Close() error {
	s.once.Do(func() {
		_ = s.reader.Close()
		close(s.closed)
	})
	return nil
}

func TestPortForwardReturnsCopyErrorAndClosesConnections(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	targetDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			targetDone <- err
			return
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			targetDone <- err
			return
		}
		if _, err := io.WriteString(conn, "response"); err != nil {
			targetDone <- err
			return
		}
		var buf [1]byte
		_, err = conn.Read(buf[:])
		targetDone <- err
	}()

	addr := listener.Addr().(*net.TCPAddr)
	s, err := NewServer(Config{
		PortForwards: []*internalversion.PortForward{{
			Name: "test", Namespace: "default",
			Spec: internalversion.PortForwardSpec{Forwards: []internalversion.Forward{{
				Ports:  []int32{8080},
				Target: &internalversion.ForwardTarget{Address: "127.0.0.1", Port: int32(addr.Port)},
			}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close() }()
	copyErr := errors.New("client output failed")
	stream := &failingPortForwardStream{reader: reader, err: copyErr, closed: make(chan struct{})}
	defer func() { _ = stream.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.PortForward(ctx, "test/default", "", 8080, stream) }()
	select {
	case err := <-done:
		if !errors.Is(err, copyErr) {
			t.Fatalf("PortForward returned %v, want %v", err, copyErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("PortForward waited for cancellation after client output failed")
	}
	select {
	case <-stream.closed:
	default:
		t.Fatal("PortForward did not close the client stream")
	}
	select {
	case err := <-targetDone:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("target Read returned %v, want EOF after connection cleanup", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("PortForward did not close the target connection")
	}
}
