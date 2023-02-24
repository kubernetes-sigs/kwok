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
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/server/portforward"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// PortForward handles a port forwarding request.
func (s *Server) PortForward(ctx context.Context, podName, podNamespace string, uid types.UID, port int32, stream io.ReadWriteCloser) error {
	defer func() {
		_ = stream.Close()
	}()

	forward, err := s.getPodsForward(podName, podNamespace, port)
	if err != nil {
		return err
	}

	if len(forward.Command) > 0 {
		return exec.Exec(exec.WithReadWriter(ctx, stream), forward.Command[0], forward.Command[1:]...)
	}

	if forward.Target != nil {
		target := forward.Target
		addr := fmt.Sprintf("%s:%d", target.Address, target.Port)
		dial, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to dial %s: %w", addr, err)
		}
		defer func() {
			_ = dial.Close()
		}()

		// TODO: remove this when upgrade to go 1.21 upgrade takes place
		buf1 := bufPool.Get().(*[]byte)
		buf2 := bufPool.Get().(*[]byte)
		defer func() {
			bufPool.Put(buf1)
			bufPool.Put(buf2)
		}()
		return tunnel(ctx, stream, dial, *buf1, *buf2)
	}

	return errors.New("no target or command")
}

var bufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

// getPortForward handles a new restful port forward request. It determines the
// pod name and uid and then calls ServePortForward.
func (s *Server) getPortForward(req *restful.Request, resp *restful.Response) {
	params := getPortForwardRequestParams(req)

	portForwardOptions, err := portforward.NewV4Options(req.Request)
	if err != nil {
		logger := log.FromContext(req.Request.Context())
		logger.Error("NewV4Options", err)
		_ = resp.WriteError(http.StatusBadRequest, err)
		return
	}

	portforward.ServePortForward(
		req.Request.Context(),
		resp.ResponseWriter,
		req.Request,
		s,
		params.podName,
		params.podNamespace,
		params.podUID,
		portForwardOptions,
		s.idleTimeout,
		s.streamCreationTimeout,
		portforward.SupportedProtocols)
}

func (s *Server) getPodsForward(podName, podNamespace string, port int32) (*internalversion.Forward, error) {
	pf, has := slices.Find(s.config.PortForwards, func(pf *internalversion.PortForward) bool {
		return pf.Name == podName && pf.Namespace == podNamespace
	})
	if has {
		forward, found := findPortInForwards(port, pf.Spec.Forwards)
		if found {
			return forward, nil
		}
	} else {
		for _, cfw := range s.config.ClusterPortForwards {
			if !cfw.Spec.Selector.Match(podName, podNamespace) {
				continue
			}

			forward, found := findPortInForwards(port, cfw.Spec.Forwards)
			if found {
				return forward, nil
			}
		}
	}

	return nil, fmt.Errorf("forward to port %d is not found", port)
}

func findPortInForwards(port int32, forwards []internalversion.Forward) (*internalversion.Forward, bool) {
	var defaultForward *internalversion.Forward
	for i, fw := range forwards {
		if len(fw.Ports) == 0 && defaultForward == nil {
			defaultForward = &forwards[i]
			continue
		}
		if slices.Contains(fw.Ports, port) {
			return &fw, true
		}
	}
	return defaultForward, defaultForward != nil
}

// tunnel create tunnels for two streams.
func tunnel(ctx context.Context, c1, c2 io.ReadWriter, buf1, buf2 []byte) error {
	errCh := make(chan error)
	go func() {
		_, err := io.CopyBuffer(c2, c1, buf1)
		errCh <- err
	}()
	go func() {
		_, err := io.CopyBuffer(c1, c2, buf2)
		errCh <- err
	}()
	select {
	case <-ctx.Done():
		// Do nothing
	case err1 := <-errCh:
		select {
		case <-ctx.Done():
			if err1 != nil {
				return err1
			}
			// Do nothing
		case err2 := <-errCh:
			if err1 != nil {
				return err1
			}
			return err2
		}
	}
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
