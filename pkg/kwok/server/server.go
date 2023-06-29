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
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/pools"
)

const (
	pprofBasePath = "/debug/pprof/"
)

// Server is a server that can serve HTTP/HTTPS requests.
type Server struct {
	typedKwokClient versioned.Interface

	enableCRDs []string

	restfulCont *restful.Container

	idleTimeout           time.Duration
	streamCreationTimeout time.Duration
	bufPool               *pools.Pool[[]byte]

	clusterPortForwards resources.Getter[[]*internalversion.ClusterPortForward]
	portForwards        resources.Getter[[]*internalversion.PortForward]
	clusterExecs        resources.Getter[[]*internalversion.ClusterExec]
	execs               resources.Getter[[]*internalversion.Exec]
	clusterLogs         resources.Getter[[]*internalversion.ClusterLogs]
	logs                resources.Getter[[]*internalversion.Logs]
	clusterAttaches     resources.Getter[[]*internalversion.ClusterAttach]
	attaches            resources.Getter[[]*internalversion.Attach]

	metrics    []*internalversion.Metric
	controller *controllers.Controller
}

// Config holds configurations needed by the server handlers.
type Config struct {
	TypedKwokClient versioned.Interface
	EnableCRDs      []string

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
func NewServer(config Config) (*Server, error) {
	container := restful.NewContainer()

	s := &Server{
		typedKwokClient:       config.TypedKwokClient,
		enableCRDs:            config.EnableCRDs,
		restfulCont:           container,
		idleTimeout:           1 * time.Hour,
		streamCreationTimeout: remotecommandconsts.DefaultStreamCreationTimeout,

		clusterPortForwards: resources.NewStaticGetter(config.ClusterPortForwards),
		portForwards:        resources.NewStaticGetter(config.PortForwards),
		clusterExecs:        resources.NewStaticGetter(config.ClusterExecs),
		execs:               resources.NewStaticGetter(config.Execs),
		clusterLogs:         resources.NewStaticGetter(config.ClusterLogs),
		logs:                resources.NewStaticGetter(config.Logs),
		clusterAttaches:     resources.NewStaticGetter(config.ClusterAttaches),
		attaches:            resources.NewStaticGetter(config.Attaches),

		metrics:    config.Metrics,
		controller: config.Controller,

		bufPool: pools.NewPool(func() []byte {
			return make([]byte, 32*1024)
		}),
	}

	return s, nil
}

func (s *Server) initWatchCRD(ctx context.Context) ([]resources.Starter, error) {
	cli := s.typedKwokClient

	starters := []resources.Starter{}

	logger := log.FromContext(ctx)

	for _, crd := range s.enableCRDs {
		switch crd {
		case v1alpha1.ClusterPortForwardKind:
			if len(s.clusterPortForwards.Get()) != 0 {
				return nil, fmt.Errorf("cluster port forwards already exists, cannot watch CRD")
			}
			clusterPortForwards := resources.NewDynamicGetter[
				[]*internalversion.ClusterPortForward,
				*v1alpha1.ClusterPortForward,
				*v1alpha1.ClusterPortForwardList,
			](
				cli.KwokV1alpha1().ClusterPortForwards(),
				func(objs []*v1alpha1.ClusterPortForward) []*internalversion.ClusterPortForward {
					return convertList(objs, func(obj *v1alpha1.ClusterPortForward) (*internalversion.ClusterPortForward, bool) {
						r, err := internalversion.ConvertToInternalClusterPortForward(obj)
						if err != nil {
							logger.Error("failed to convert to internal cluster port forward", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, clusterPortForwards)
			s.clusterPortForwards = clusterPortForwards
		case v1alpha1.PortForwardKind:
			if len(s.portForwards.Get()) != 0 {
				return nil, fmt.Errorf("port forwards already exists, cannot watch CRD")
			}
			portForwards := resources.NewDynamicGetter[
				[]*internalversion.PortForward,
				*v1alpha1.PortForward,
				*v1alpha1.PortForwardList,
			](
				cli.KwokV1alpha1().PortForwards(""),
				func(objs []*v1alpha1.PortForward) []*internalversion.PortForward {
					return convertList(objs, func(obj *v1alpha1.PortForward) (*internalversion.PortForward, bool) {
						r, err := internalversion.ConvertToInternalPortForward(obj)
						if err != nil {
							logger.Error("failed to convert to internal port forward", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, portForwards)
			s.portForwards = portForwards
		case v1alpha1.ClusterExecKind:
			if len(s.clusterExecs.Get()) != 0 {
				return nil, fmt.Errorf("cluster execs already exists, cannot watch CRD")
			}
			clusterExecs := resources.NewDynamicGetter[
				[]*internalversion.ClusterExec,
				*v1alpha1.ClusterExec,
				*v1alpha1.ClusterExecList,
			](
				cli.KwokV1alpha1().ClusterExecs(),
				func(objs []*v1alpha1.ClusterExec) []*internalversion.ClusterExec {
					return convertList(objs, func(obj *v1alpha1.ClusterExec) (*internalversion.ClusterExec, bool) {
						r, err := internalversion.ConvertToInternalClusterExec(obj)
						if err != nil {
							logger.Error("failed to convert to internal cluster exec", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, clusterExecs)
			s.clusterExecs = clusterExecs
		case v1alpha1.ExecKind:
			if len(s.execs.Get()) != 0 {
				return nil, fmt.Errorf("execs already exists, cannot watch CRD")
			}
			execs := resources.NewDynamicGetter[
				[]*internalversion.Exec,
				*v1alpha1.Exec,
				*v1alpha1.ExecList,
			](
				cli.KwokV1alpha1().Execs(""),
				func(objs []*v1alpha1.Exec) []*internalversion.Exec {
					return convertList(objs, func(obj *v1alpha1.Exec) (*internalversion.Exec, bool) {
						r, err := internalversion.ConvertToInternalExec(obj)
						if err != nil {
							logger.Error("failed to convert to internal exec", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, execs)
			s.execs = execs
		case v1alpha1.ClusterLogsKind:
			if len(s.clusterLogs.Get()) != 0 {
				return nil, fmt.Errorf("cluster logs already exists, cannot watch CRD")
			}
			clusterLogs := resources.NewDynamicGetter[
				[]*internalversion.ClusterLogs,
				*v1alpha1.ClusterLogs,
				*v1alpha1.ClusterLogsList,
			](
				cli.KwokV1alpha1().ClusterLogs(),
				func(objs []*v1alpha1.ClusterLogs) []*internalversion.ClusterLogs {
					return convertList(objs, func(obj *v1alpha1.ClusterLogs) (*internalversion.ClusterLogs, bool) {
						r, err := internalversion.ConvertToInternalClusterLogs(obj)
						if err != nil {
							logger.Error("failed to convert to internal cluster logs", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, clusterLogs)
			s.clusterLogs = clusterLogs
		case v1alpha1.LogsKind:
			if len(s.logs.Get()) != 0 {
				return nil, fmt.Errorf("logs already exists, cannot watch CRD")
			}
			logs := resources.NewDynamicGetter[
				[]*internalversion.Logs,
				*v1alpha1.Logs,
				*v1alpha1.LogsList,
			](
				cli.KwokV1alpha1().Logs(""),
				func(objs []*v1alpha1.Logs) []*internalversion.Logs {
					return convertList(objs, func(obj *v1alpha1.Logs) (*internalversion.Logs, bool) {
						r, err := internalversion.ConvertToInternalLogs(obj)
						if err != nil {
							logger.Error("failed to convert to internal logs", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, logs)
			s.logs = logs
		case v1alpha1.ClusterAttachKind:
			if len(s.clusterAttaches.Get()) != 0 {
				return nil, fmt.Errorf("cluster attaches already exists, cannot watch CRD")
			}
			clusterAttaches := resources.NewDynamicGetter[
				[]*internalversion.ClusterAttach,
				*v1alpha1.ClusterAttach,
				*v1alpha1.ClusterAttachList,
			](
				cli.KwokV1alpha1().ClusterAttaches(),
				func(objs []*v1alpha1.ClusterAttach) []*internalversion.ClusterAttach {
					return convertList(objs, func(obj *v1alpha1.ClusterAttach) (*internalversion.ClusterAttach, bool) {
						r, err := internalversion.ConvertToInternalClusterAttach(obj)
						if err != nil {
							logger.Error("failed to convert to internal cluster attach", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, clusterAttaches)
			s.clusterAttaches = clusterAttaches
		case v1alpha1.AttachKind:
			if len(s.attaches.Get()) != 0 {
				return nil, fmt.Errorf("attaches already exists, cannot watch CRD")
			}
			attaches := resources.NewDynamicGetter[
				[]*internalversion.Attach,
				*v1alpha1.Attach,
				*v1alpha1.AttachList,
			](
				cli.KwokV1alpha1().Attaches(""),
				func(objs []*v1alpha1.Attach) []*internalversion.Attach {
					return convertList(objs, func(obj *v1alpha1.Attach) (*internalversion.Attach, bool) {
						r, err := internalversion.ConvertToInternalAttach(obj)
						if err != nil {
							logger.Error("failed to convert to internal attach", err, "obj", obj)
							return nil, false
						}
						return r, true
					})
				},
			)
			starters = append(starters, attaches)
		}
	}
	return starters, nil
}

func convertList[S ~[]T, T any, O any](s S, f func(T) (O, bool)) []O {
	out := make([]O, 0, len(s))
	for _, d := range s {
		o, ok := f(d)
		if !ok {
			continue
		}
		out = append(out, o)
	}
	return out
}

func getHandlerForDisabledEndpoint(errorMessage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, errorMessage, http.StatusMethodNotAllowed)
	}
}

// Run runs the specified Server.
// This should never exit.
func (s *Server) Run(ctx context.Context, address string, certFile, privateKeyFile string) error {
	if len(s.enableCRDs) != 0 {
		starters, err := s.initWatchCRD(ctx)
		if err != nil {
			return fmt.Errorf("init enable crd: %w", err)
		}
		for _, starter := range starters {
			if err := starter.Start(ctx); err != nil {
				return fmt.Errorf("start crd getter: %w", err)
			}
		}
	}
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
