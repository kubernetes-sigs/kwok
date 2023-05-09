/*
Copyright 2022 The Kubernetes Authors.

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

package net

import (
	"context"
	"fmt"
	"net"
)

var (
	errGetUnusedPort        = fmt.Errorf("unable to get an unused port")
	lastUsedPort     uint32 = 32767
)

// GetUnusedPort returns an unused port on the local machine.
func GetUnusedPort(ctx context.Context) (uint32, error) {
	for lastUsedPort > 10000 && ctx.Err() == nil {
		lastUsedPort--
		if isPortUnused(lastUsedPort) {
			return lastUsedPort, nil
		}
	}

	return 0, errGetUnusedPort
}

func isPortUnused(port uint32) bool {
	return isHostPortUnused(LocalAddress, port) && isHostPortUnused("", port)
}

func isHostPortUnused(host string, port uint32) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%d", host, port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
