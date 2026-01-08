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

package compose

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// PortForward expose the port of the component
func (c *Cluster) PortForward(ctx context.Context, name string, portOrName string, hostPort uint32) (cancel func(), retErr error) {
	kwokController, err := c.GetComponent(ctx, consts.ComponentKwokController)
	if err != nil {
		return nil, err
	}

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

	logger := log.FromContext(ctx)

	tempContainerName := "temp-port-forward-proxy-" + format.String(time.Now().Unix())

	args := []string{
		"run",
		"--rm",
		"-i",
	}

	if !c.isAppleContainer {
		args = append(args, "--pull=never")
	}

	args = append(args,
		"--network="+c.networkName(),
		"--name="+tempContainerName,
		"--entrypoint=/bin/sh",
	)

	args = append(args, c.labelArgs()...)
	args = append(args, kwokController.Image)

	r, w := io.Pipe()
	command, err := exec.Command(exec.WithWait(exec.WithIOStreams(ctx, exec.IOStreams{In: r}), false),
		c.runtime, args...)
	if err != nil {
		return nil, fmt.Errorf("running command: %w", err)
	}

	cleanTempContainer := func() {
		_, _ = w.Write([]byte("exit\n"))
		_ = command.Cancel()

		if c.runtime == consts.RuntimeTypeNerdctl ||
			c.runtime == consts.RuntimeTypeLima ||
			c.runtime == consts.RuntimeTypeFinch {
			_, err := exec.Command(context.Background(), c.runtime, "rm", "--force", tempContainerName)
			if err != nil {
				logger.Error("Remove temporary port-forward container", err)
			}
		}
	}

	defer func() {
		if retErr != nil {
			cleanTempContainer()
		}
	}()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", hostPort))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = listener.Close()
		}
	}()

	cancel = func() {
		_ = listener.Close()
		cleanTempContainer()
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
				err := c.Exec(exec.WithReadWriter(ctx, conn),
					c.runtime, "exec", "-i", tempContainerName,
					"nc", c.Name()+"-"+name, format.String(targetPort))
				if err != nil {
					logger.Warn("failed tunneling port", "err", err)
				}
			}()
		}
	}()

	return cancel, nil
}
