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

package binary

import (
	"context"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addEtcd(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the etcd
	etcdPath, err := c.EnsureBinary(ctx, consts.ComponentEtcd, conf.EtcdBinary)
	if err != nil {
		return err
	}

	etcdVersion, err := c.ParseVersionFromBinary(ctx, etcdPath)
	if err != nil {
		return err
	}

	otlpGrpcAddress := ""
	if conf.JaegerOtlpGrpcPort != 0 {
		otlpGrpcAddress = utilsnet.LocalAddress + ":" + format.String(conf.JaegerOtlpGrpcPort)
	}

	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Binary:           etcdPath,
		Version:          etcdVersion,
		BindAddress:      conf.BindAddress,
		DataPath:         env.etcdDataPath,
		Port:             conf.EtcdPort,
		PeerPort:         conf.EtcdPeerPort,
		Verbosity:        env.verbosity,
		QuotaBackendSize: conf.EtcdQuotaBackendSize,
		OtlpGrpcAddress:  otlpGrpcAddress,
	})
	if err != nil {
		return err
	}
	env.kwokctlConfig.Components = append(env.kwokctlConfig.Components, etcdComponent)
	return nil
}
