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
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	toolsportforward "k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/kubelet/pkg/cri/streaming/portforward"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// PortForward handles a port forwarding request.
func (s *Server) PortForward(ctx context.Context, name string, uid types.UID, port int32, stream io.ReadWriteCloser) error {
	defer func() {
		_ = stream.Close()
	}()

	pod := strings.Split(name, "/")
	if len(pod) != 2 {
		return fmt.Errorf("invalid pod name %q", name)
	}
	podName, podNamespace := pod[0], pod[1]

	forward, err := getPodsForward(s.portForwards.Get(), s.clusterPortForwards.Get(), podName, podNamespace, port)
	if err != nil {
		return err
	}

	if m := forward.Mapping; m != nil {
		return s.portForwardMappingToContainer(ctx, m.Namespace, m.Name, port, stream)
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
		buf1 := s.bufPool.Get()
		buf2 := s.bufPool.Get()
		defer func() {
			s.bufPool.Put(buf1)
			s.bufPool.Put(buf2)
		}()
		return utilsnet.Tunnel(ctx, stream, dial, buf1, buf2)
	}

	return errors.New("no target or command")
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
		resp.ResponseWriter,
		req.Request,
		s,
		params.podName+"/"+params.podNamespace,
		params.podUID,
		portForwardOptions,
		s.idleTimeout,
		s.streamCreationTimeout,
		portforward.SupportedProtocols,
	)
}

func getPodsForward(rules []*internalversion.PortForward, clusterRules []*internalversion.ClusterPortForward, podName, podNamespace string, port int32) (*internalversion.Forward, error) {
	pf, has := slices.Find(rules, func(pf *internalversion.PortForward) bool {
		return pf.Name == podName && pf.Namespace == podNamespace
	})
	if has {
		forward, found := findPortInForwards(port, pf.Spec.Forwards)
		if found {
			return forward, nil
		}
		return nil, fmt.Errorf("forward not found for port %d in pod %q", port, log.KRef(podNamespace, podName))
	}

	for _, cfw := range clusterRules {
		if !cfw.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		forward, found := findPortInForwards(port, cfw.Spec.Forwards)
		if found {
			return forward, nil
		}
	}
	return nil, fmt.Errorf("no forward found for port %d in pod %q", port, log.KRef(podNamespace, podName))
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

// portForwardMappingToContainer returns port forward for a container in a pod with mapping.
func (s *Server) portForwardMappingToContainer(ctx context.Context, namespace, name string, port int32, stream io.ReadWriteCloser) error {
	defer func() {
		_ = stream.Close()
	}()

	req := s.typedClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(namespace).
		SubResource("portforward")

	req.VersionedParams(&corev1.PodPortForwardOptions{
		Ports: []int32{port},
	}, scheme.ParameterCodec)

	roundTripper, upgrader, err := spdy.RoundTripperFor(s.restConfig)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

	streamConn, _, err := dialer.Dial(toolsportforward.PortForwardProtocolV1Name)
	if err != nil {
		return err
	}

	defer func() {
		_ = streamConn.Close()
	}()

	requestID := 0

	// create error stream
	headers := http.Header{}
	headers.Set(corev1.StreamType, corev1.StreamTypeError)
	headers.Set(corev1.PortHeader, fmt.Sprintf("%d", port))
	headers.Set(corev1.PortForwardRequestIDHeader, strconv.Itoa(requestID))
	errorStream, err := streamConn.CreateStream(headers)
	if err != nil {
		return err
	}
	// we're not writing to this stream
	_ = errorStream.Close()
	defer streamConn.RemoveStreams(errorStream)

	errorChan := make(chan error)
	go func() {
		message, err := io.ReadAll(errorStream)
		switch {
		case err != nil:
			errorChan <- fmt.Errorf("error reading from error stream for port: %w", err)
		case len(message) > 0:
			errorChan <- fmt.Errorf("an error occurred forwarding: %s", string(message))
		}
		close(errorChan)
	}()

	// create data stream
	headers.Set(corev1.StreamType, corev1.StreamTypeData)
	dataStream, err := streamConn.CreateStream(headers)
	if err != nil {
		return fmt.Errorf("error creating forwarding stream for port: %w", err)
	}
	defer streamConn.RemoveStreams(dataStream)

	localError := make(chan struct{})
	remoteDone := make(chan struct{})

	go func() {
		// Copy from the remote side to the local port.
		if _, err := io.Copy(stream, dataStream); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			runtime.HandleError(fmt.Errorf("error copying from remote stream to local connection: %w", err))
		}

		// inform the select below that the remote copy is done
		close(remoteDone)
	}()

	go func() {
		// inform server we're not sending any more data after copy unblocks
		defer func() {
			_ = dataStream.Close()
		}()

		// Copy from the local port to the remote side.
		if _, err := io.Copy(dataStream, stream); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			runtime.HandleError(fmt.Errorf("error copying from local connection to remote stream: %w", err))
			// break out of the select below without waiting for the other copy to finish
			close(localError)
		}
	}()

	// wait for either a local->remote error or for copying from remote->local to finish
	select {
	case <-remoteDone:
	case <-localError:
	case <-ctx.Done():
	}

	// always expect something on errorChan (it may be nil)
	err = <-errorChan
	if err != nil {
		return err
	}

	return nil
}
