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
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// BuildEtcdComponentConfig is the configuration for building an etcd component.
type BuildEtcdComponentConfig struct {
	Binary    string
	Image     string
	Version   version.Version
	DataPath  string
	Workdir   string
	Address   string
	Port      uint32
	PeerPort  uint32
	ExtraArgs []string
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
	if conf.Address == "" {
		conf.Address = publicAddress
	}

	var volumes []internalversion.Volume
	var ports []internalversion.Port

	etcdArgs := []string{
		"--name=node0",
		"--auto-compaction-retention=1",
		"--quota-backend-bytes=8589934592",
	}
	etcdArgs = append(etcdArgs, conf.ExtraArgs...)

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
			"--initial-advertise-peer-urls=http://"+conf.Address+":2380",
			"--listen-peer-urls=http://"+conf.Address+":2380",
			"--advertise-client-urls=http://"+conf.Address+":2379",
			"--listen-client-urls=http://"+conf.Address+":2379",
			"--initial-cluster=node0=http://"+conf.Address+":2380",
		)
	} else {
		etcdPeerPortStr := format.String(conf.PeerPort)
		etcdClientPortStr := format.String(conf.Port)
		etcdArgs = append(etcdArgs,
			"--data-dir="+conf.DataPath,
			"--initial-advertise-peer-urls=http://"+conf.Address+":"+etcdPeerPortStr,
			"--listen-peer-urls=http://"+conf.Address+":"+etcdPeerPortStr,
			"--advertise-client-urls=http://"+conf.Address+":"+etcdClientPortStr,
			"--listen-client-urls=http://"+conf.Address+":"+etcdClientPortStr,
			"--initial-cluster=node0=http://"+conf.Address+":"+etcdPeerPortStr,
		)
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
