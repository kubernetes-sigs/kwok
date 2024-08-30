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

package components

import (
	"runtime"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildEtcdComponentConfig is the configuration for building an etcd component.
type BuildEtcdComponentConfig struct {
	Runtime     string
	Binary      string
	Image       string
	ProjectName string
	Version     version.Version
	DataPath    string
	Workdir     string
	BindAddress string
	Port        uint32
	PeerPort    uint32
	Verbosity   log.Level
}

// BuildEtcdComponent builds an etcd component.
func BuildEtcdComponent(conf BuildEtcdComponentConfig) (component internalversion.Component, err error) {
	var volumes []internalversion.Volume
	var ports []internalversion.Port

	etcdArgs := []string{
		"--name=node0",
		"--auto-compaction-retention=1",
		"--quota-backend-bytes=8589934592",
	}

	var metric *internalversion.ComponentMetric

	if GetRuntimeMode(conf.Runtime) != RuntimeModeNative {
		// TODO: use a volume for the data path
		// volumes = append(volumes,
		//	internalversion.Volume{
		//		HostPath:  conf.DataPath,
		//		MountPath: "/etcd-data",
		//	},
		//)
		etcdArgs = append(etcdArgs,
			"--data-dir=/etcd-data",
		)

		ports = append(
			ports,
			internalversion.Port{
				Name:     "peer-http",
				HostPort: conf.PeerPort,
				Port:     2380,
				Protocol: internalversion.ProtocolTCP,
			},
			internalversion.Port{
				Name:     "http",
				HostPort: conf.Port,
				Port:     2379,
				Protocol: internalversion.ProtocolTCP,
			},
		)
		etcdArgs = append(etcdArgs,
			"--initial-advertise-peer-urls=http://"+conf.BindAddress+":2380",
			"--listen-peer-urls=http://"+conf.BindAddress+":2380",
			"--advertise-client-urls=http://"+conf.BindAddress+":2379",
			"--listen-client-urls=http://"+conf.BindAddress+":2379",
			"--initial-cluster=node0=http://"+conf.BindAddress+":2380",
		)

		metric = &internalversion.ComponentMetric{
			Scheme: "http",
			Host:   conf.ProjectName + "-" + consts.ComponentEtcd + ":2379",
			Path:   "/metrics",
		}
	} else {
		etcdPeerPortStr := format.String(conf.PeerPort)
		etcdClientPortStr := format.String(conf.Port)

		ports = append(
			ports,
			internalversion.Port{
				Name:     "peer-http",
				HostPort: 0,
				Port:     conf.PeerPort,
				Protocol: internalversion.ProtocolTCP,
			},
			internalversion.Port{
				Name:     "http",
				HostPort: 0,
				Port:     conf.Port,
				Protocol: internalversion.ProtocolTCP,
			},
		)

		etcdArgs = append(etcdArgs,
			"--data-dir="+conf.DataPath,
			"--initial-advertise-peer-urls=http://"+conf.BindAddress+":"+etcdPeerPortStr,
			"--listen-peer-urls=http://"+conf.BindAddress+":"+etcdPeerPortStr,
			"--advertise-client-urls=http://"+conf.BindAddress+":"+etcdClientPortStr,
			"--listen-client-urls=http://"+conf.BindAddress+":"+etcdClientPortStr,
			"--initial-cluster=node0=http://"+conf.BindAddress+":"+etcdPeerPortStr,
		)

		metric = &internalversion.ComponentMetric{
			Scheme: "http",
			Host:   net.LocalAddress + ":" + etcdClientPortStr,
			Path:   "/metrics",
		}
	}

	if conf.Version.GTE(version.NewVersion(3, 4, 0)) {
		if conf.Verbosity != log.LevelInfo {
			etcdArgs = append(etcdArgs, "--log-level="+log.ToLogSeverityLevel(conf.Verbosity))
		}
	} else {
		if conf.Verbosity <= log.LevelDebug {
			etcdArgs = append(etcdArgs, "--debug")
		}
	}

	envs := []internalversion.Env{}
	if runtime.GOARCH != "amd64" {
		envs = append(envs, internalversion.Env{
			Name:  "ETCD_UNSUPPORTED_ARCH",
			Value: runtime.GOARCH,
		})
	}

	return internalversion.Component{
		Name:    consts.ComponentEtcd,
		Version: conf.Version.String(),
		Volumes: volumes,
		Command: []string{consts.ComponentEtcd},
		Args:    etcdArgs,
		Binary:  conf.Binary,
		Ports:   ports,
		Metric:  metric,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
		Envs:    envs,
	}, nil
}
