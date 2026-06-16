/*
Copyright 2026 The Kubernetes Authors.

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
)

// EtcdctlInCluster implements the ectdctl subcommand
func (c *Cluster) EtcdctlInCluster(ctx context.Context, args ...string) error {
	etcdContainerName := c.Name() + "-etcd"

	// If using versions earlier than v3.4, set `ETCDCTL_API=3` to use v3 API.
	args = append([]string{"exec", "--env=ETCDCTL_API=3", "-i", etcdContainerName, "etcdctl"}, args...)
	return c.Exec(ctx, c.runtime, args...)
}
