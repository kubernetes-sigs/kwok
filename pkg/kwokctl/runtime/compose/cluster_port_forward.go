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
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
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
		port, ok := utilsslices.Find(component.Ports, func(port internalversion.Port) bool {
			return port.Name == portOrName && port.Protocol == internalversion.ProtocolTCP
		})
		if !ok {
			return nil, fmt.Errorf("port %q not found", portOrName)
		}
		targetPort = uint64(port.Port)
	}

	logger := log.FromContext(ctx)

	tempContainerName := "temp-port-forward-proxy-" + format.String(time.Now().Unix())
	labelArgs := c.labelArgs()
	args := make([]string, 0, 7+len(labelArgs)+1)
	args = append(args,
		"run",
		"--rm",
		"-i",
		"--pull=never",
		"--network="+c.Name(),
		"--name="+tempContainerName,
		"--entrypoint=/bin/sh",
	)

	args = append(args, labelArgs...)
	args = append(args, kwokController.Image)

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	command, err := utilsexec.Command(utilsexec.WithWait(utilsexec.WithIOStreams(ctx, utilsexec.IOStreams{In: inR, Out: outW, ErrOut: outW}), false),
		c.runtime, args...)
	if err != nil {
		return nil, fmt.Errorf("running command: %w", err)
	}

	_, _ = inW.Write([]byte("pwd\n"))

	var buf [16]byte
	_, err = outR.Read(buf[:])
	if err != nil {
		return nil, fmt.Errorf("starting temporary port-forward container: %w", err)
	}

	cleanTempContainer := func() {
		_, _ = inW.Write([]byte("exit\n"))
		_ = command.Cancel()

		if c.runtime == consts.RuntimeTypeNerdctl ||
			c.runtime == consts.RuntimeTypeLima ||
			c.runtime == consts.RuntimeTypeFinch {
			_, err := utilsexec.Command(context.Background(), c.runtime, "rm", "--force", tempContainerName)
			if err != nil {
				logger.Error("Remove temporary port-forward container",
					"err", err,
				)
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

	ctx, ctxCancel := context.WithCancel(ctx)

	cancel = func() {
		ctxCancel()
		_ = listener.Close()
		cleanTempContainer()
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error("accepting connection",
					"err", err,
				)
				return
			}

			go func() {
				defer func() {
					_ = conn.Close()
				}()
				err := c.Exec(utilsexec.WithReadWriter(ctx, conn),
					c.runtime, "exec", "-i", tempContainerName,
					"nc", c.Name()+"-"+name, format.String(targetPort))
				if err != nil {
					logger.Warn("failed tunneling port",
						"err", err,
					)
				}
			}()
		}
	}()

	return cancel, nil
}
