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
	"sigs.k8s.io/kwok/pkg/utils/net"
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
	jaegerArgs := []string{"--collector.otlp.enabled=true"}

	var volumes []internalversion.Volume
	var ports []internalversion.Port

	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		ports = []internalversion.Port{
			{
				HostPort: conf.Port,
				Port:     16686,
			},
		}
		jaegerArgs = append(jaegerArgs,
			"--query.http-server.host-port="+conf.BindAddress+":16686",
		)
	} else {
		jaegerArgs = append(jaegerArgs,
			"--query.http-server.host-port="+conf.BindAddress+":"+format.String(conf.Port),
			"--collector.otlp.grpc.host-port="+net.LocalAddress+":"+format.String(conf.OtlpGrpcPort),
		)
	}

	if conf.Verbosity != log.LevelInfo {
		jaegerArgs = append(jaegerArgs, "--log-level="+log.ToLogSeverityLevel(conf.Verbosity))
	}

	return internalversion.Component{
		Name:    consts.ComponentJaeger,
		Version: conf.Version.String(),
		Ports:   ports,
		Volumes: volumes,
		Args:    jaegerArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
