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

	"sigs.k8s.io/kwok/pkg/kwokctl/components"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
)

func (c *Cluster) addEtcd(ctx context.Context, env *env) (err error) {
	conf := &env.kwokctlConfig.Options

	// Configure the etcd
	err = c.EnsureImage(ctx, c.runtime, conf.EtcdImage)
	if err != nil {
		return err
	}
	etcdVersion, err := c.ParseVersionFromImage(ctx, c.runtime, conf.EtcdImage, "etcd")
	if err != nil {
		return err
	}

	otlpGrpcAddress := ""
	if conf.JaegerPort != 0 {
		otlpGrpcAddress = c.Name() + "-jaeger:4317"
	}

	etcdComponent, err := components.BuildEtcdComponent(components.BuildEtcdComponentConfig{
		Runtime:          conf.Runtime,
		ProjectName:      c.Name(),
		Workdir:          env.workdir,
		Image:            conf.EtcdImage,
		Version:          etcdVersion,
		BindAddress:      utilsnet.PublicAddress,
		Port:             conf.EtcdPort,
		DataPath:         env.etcdDataPath,
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
