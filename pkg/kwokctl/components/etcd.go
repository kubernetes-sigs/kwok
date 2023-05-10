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
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildEtcdComponentConfig is the configuration for building an etcd component.
type BuildEtcdComponentConfig struct {
	Binary       string
	Image        string
	Version      version.Version
	DataPath     string
	Workdir      string
	BindAddress  string
	Port         uint32
	PeerPort     uint32
	Verbosity    log.Level
	ExtraArgs    []internalversion.ExtraArgs
	ExtraVolumes []internalversion.Volume
}

// BuildEtcdComponent builds an etcd component.
func BuildEtcdComponent(conf BuildEtcdComponentConfig) (component internalversion.Component, err error) {
	exposePeerPort := true
	if conf.PeerPort == 0 {
		conf.PeerPort = 2380
		exposePeerPort = false
	}
	exposePort := true
	if conf.Port == 0 {
		conf.Port = 2379
		exposePort = false
	}

	var volumes []internalversion.Volume
	volumes = append(volumes, conf.ExtraVolumes...)
	var ports []internalversion.Port

	etcdArgs := []string{
		"--name=node0",
		"--auto-compaction-retention=1",
		"--quota-backend-bytes=8589934592",
	}
	etcdArgs = append(etcdArgs, extraArgsToStrings(conf.ExtraArgs)...)

	inContainer := conf.Image != ""
	if inContainer {
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

		if exposePeerPort {
			ports = append(
				ports,
				internalversion.Port{
					HostPort: conf.PeerPort,
					Port:     2380,
				},
			)
		}
		if exposePort {
			ports = append(
				ports,
				internalversion.Port{
					HostPort: conf.Port,
					Port:     2379,
				},
			)
		}
		etcdArgs = append(etcdArgs,
			"--initial-advertise-peer-urls=http://"+conf.BindAddress+":2380",
			"--listen-peer-urls=http://"+conf.BindAddress+":2380",
			"--advertise-client-urls=http://"+conf.BindAddress+":2379",
			"--listen-client-urls=http://"+conf.BindAddress+":2379",
			"--initial-cluster=node0=http://"+conf.BindAddress+":2380",
		)
	} else {
		etcdPeerPortStr := format.String(conf.PeerPort)
		etcdClientPortStr := format.String(conf.Port)
		etcdArgs = append(etcdArgs,
			"--data-dir="+conf.DataPath,
			"--initial-advertise-peer-urls=http://"+conf.BindAddress+":"+etcdPeerPortStr,
			"--listen-peer-urls=http://"+conf.BindAddress+":"+etcdPeerPortStr,
			"--advertise-client-urls=http://"+conf.BindAddress+":"+etcdClientPortStr,
			"--listen-client-urls=http://"+conf.BindAddress+":"+etcdClientPortStr,
			"--initial-cluster=node0=http://"+conf.BindAddress+":"+etcdPeerPortStr,
		)
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

	return internalversion.Component{
		Name:    "etcd",
		Version: conf.Version.String(),
		Volumes: volumes,
		Command: []string{"etcd"},
		Args:    etcdArgs,
		Binary:  conf.Binary,
		Ports:   ports,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
