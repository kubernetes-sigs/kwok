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

package kind

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
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

			go func() {
				defer func() {
					_ = conn.Close()
				}()

				command := fmt.Sprintf(`{ cat <&3 & cat >&3; } 3<> /dev/tcp/127.0.0.1/%d`, targetPort)
				err := c.Exec(exec.WithReadWriter(ctx, conn),
					c.runtime, "exec", "-i", c.getClusterName(),
					"bash", "-c", command)
				if err != nil {
					logger.Warn("failed tunneling port", "err", err)
				}
			}()
		}
	}()

	return cancel, nil
}
