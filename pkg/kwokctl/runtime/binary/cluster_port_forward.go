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

package binary

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// PortForward expose the port of the component
func (c *Cluster) PortForward(ctx context.Context, name string, portOrName string, hostPort uint32) (cancel func(), retErr error) {
	targetPort, err := strconv.ParseUint(portOrName, 0, 0)
	if err != nil {
		component, err := c.GetComponent(ctx, name)
		if err != nil {
			return nil, err
		}
		port, ok := slices.Find(component.Ports, func(port internalversion.Port) bool {
			return port.Name == portOrName && port.Protocol == internalversion.ProtocolTCP
		})
		if !ok {
			return nil, fmt.Errorf("port %q not found", portOrName)
		}
		targetPort = uint64(port.Port)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", hostPort))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = listener.Close()
		}
	}()

	logger := log.FromContext(ctx)
	cancel = func() {
		_ = listener.Close()
	}

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Error("accepting connection", err)
				return
			}

			target, err := net.Dial("tcp", fmt.Sprintf(":%d", targetPort))
			if err != nil {
				_ = conn.Close()
				logger.Error("failed to connect", err, "port", targetPort)
				return
			}
			go func() {
				defer func() {
					_ = target.Close()
					_ = conn.Close()
				}()
				err = utilsnet.Tunnel(ctx, conn, target, nil, nil)
				if err != nil {
					logger.Warn("failed tunneling port", "err", err)
				}
			}()
		}
	}()

	return cancel, nil
}
