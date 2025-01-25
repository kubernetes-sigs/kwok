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

package runtime

import (
	"context"

	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) Runtime(ctx context.Context) string {
	config, err := c.Config(ctx)
	if err != nil {
		return ""
	}
	conf := &config.Options

	return conf.Runtime
}

func (c *Cluster) Mode(ctx context.Context) string {
	return components.GetRuntimeMode(c.Runtime(ctx))
}

func (c *Cluster) ComponentAddress(ctx context.Context, name string) string {
	switch c.Mode(ctx) {
	case components.RuntimeModeContainer:
		return c.Name() + "-" + name
	default:
		return net.LocalAddress
	}
}
