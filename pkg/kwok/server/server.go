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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/wzshiming/cmux"
	"github.com/wzshiming/cmux/pattern"
	remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/pools"
)

const (
	pprofBasePath = "/debug/pprof/"
)

// Server is a server that can serve HTTP/HTTPS requests.
type Server struct {
	restfulCont *restful.Container

	idleTimeout           time.Duration
	streamCreationTimeout time.Duration
	config                Config
	bufPool               *pools.Pool[[]byte]
}

// Config holds configurations needed by the server handlers.
type Config struct {
	ClusterPortForwards []*internalversion.ClusterPortForward
	PortForwards        []*internalversion.PortForward
	ClusterExecs        []*internalversion.ClusterExec
	Execs               []*internalversion.Exec
	ClusterLogs         []*internalversion.ClusterLogs
	Logs                []*internalversion.Logs
	ClusterAttaches     []*internalversion.ClusterAttach
	Attaches            []*internalversion.Attach
	Metrics             []*internalversion.Metric
	Controller          *controllers.Controller
}

// NewServer creates a new Server.
func NewServer(config Config) *Server {
	container := restful.NewContainer()
	return &Server{
		restfulCont:           container,
		idleTimeout:           1 * time.Hour,
		streamCreationTimeout: remotecommandconsts.DefaultStreamCreationTimeout,
		config:                config,
		bufPool: pools.NewPool(func() []byte {
			return make([]byte, 32*1024)
		}),
	}
}

func getHandlerForDisabledEndpoint(errorMessage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errorMessage, http.StatusMethodNotAllowed)
	}
}

// Run runs the specified Server.
// This should never exit.
func (s *Server) Run(ctx context.Context, address string, certFile, privateKeyFile string) error {
	logger := log.FromContext(ctx)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	muxListener := cmux.NewMuxListener(listener)
	tlsListener, err := muxListener.MatchPrefix(pattern.Pattern[pattern.TLS]...)
	if err != nil {
		return fmt.Errorf("match tls listener: %w", err)
	}
	unmatchedListener, err := muxListener.Unmatched()
	if err != nil {
		return fmt.Errorf("unmatched listener: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)

	if certFile != "" && privateKeyFile != "" {
		go func() {
			logger.Info("Starting HTTPS server",
				"address", address,
				"cert", certFile,
				"key", privateKeyFile,
			)
			svc := &http.Server{
				ReadHeaderTimeout: 5 * time.Second,
				BaseContext: func(_ net.Listener) context.Context {
					return ctx
				},
				Addr:    address,
				Handler: s.restfulCont,
			}
			err = svc.ServeTLS(tlsListener, certFile, privateKeyFile)
			if err != nil {
				errCh <- fmt.Errorf("serve https: %w", err)
			}
		}()
	}

	go func() {
		logger.Info("Starting HTTP server",
			"address", address,
		)
		svc := &http.Server{
			ReadHeaderTimeout: 5 * time.Second,
			BaseContext: func(_ net.Listener) context.Context {
				return ctx
			},
			Addr:    address,
			Handler: s.restfulCont,
		}
		err = svc.Serve(unmatchedListener)
		if err != nil {
			errCh <- fmt.Errorf("serve http: %w", err)
		}
	}()

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return err
}
