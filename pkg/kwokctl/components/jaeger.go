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

package components

import (
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildJaegerComponentConfig is the configuration for building a jaeger component.
type BuildJaegerComponentConfig struct {
	Runtime      string
	Binary       string
	Image        string
	Version      version.Version
	Workdir      string
	BindAddress  string
	Port         uint32
	OtlpGrpcPort uint32
	Verbosity    log.Level
}

// BuildJaegerComponent builds a jaeger component.
func BuildJaegerComponent(conf BuildJaegerComponentConfig) (component internalversion.Component, err error) {
	var args []string
	var volumes []internalversion.Volume
	var ports []internalversion.Port

	args = append(args,
		"--collector.otlp.enabled=true",
	)

	if GetRuntimeNetwork(conf.Runtime) != RuntimeNetworkHost {
		ports = append(
			ports,
			internalversion.Port{
				Name:     "http",
				HostPort: conf.Port,
				Port:     16686,
				Protocol: internalversion.ProtocolTCP,
			},
			internalversion.Port{
				Name:     "otlp-grpc",
				HostPort: conf.OtlpGrpcPort,
				Port:     4317,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		args = append(args,
			"--query.http-server.host-port="+conf.BindAddress+":16686",
			"--collector.otlp.grpc.host-port="+conf.BindAddress+":4317",
		)
	} else {
		ports = append(
			ports,
			internalversion.Port{
				Name:     "http",
				HostPort: 0,
				Port:     conf.Port,
				Protocol: internalversion.ProtocolTCP,
			},
			internalversion.Port{
				Name:     "otlp-grpc",
				HostPort: 0,
				Port:     conf.OtlpGrpcPort,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		args = append(args,
			"--query.http-server.host-port="+conf.BindAddress+":"+format.String(conf.Port),
			"--collector.otlp.grpc.host-port="+utilsnet.LocalAddress+":"+format.String(conf.OtlpGrpcPort),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		args = append(args, "--log-level="+log.ToLogSeverityLevel(conf.Verbosity))
	}

	return internalversion.Component{
		Name:    consts.ComponentJaeger,
		Version: conf.Version.String(),
		Ports:   ports,
		Volumes: volumes,
		Args:    args,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
