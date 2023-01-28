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
	Binary   string
	Image    string
	Version  version.Version
	DataPath string
	Workdir  string
	Address  string
	Port     uint32
	PeerPort uint32
}

// BuildEtcdComponent builds an etcd component.
func BuildEtcdComponent(conf BuildEtcdComponentConfig) (component internalversion.Component, err error) {
	if conf.PeerPort == 0 {
		conf.PeerPort = 2380
	}
	if conf.Port == 0 {
		conf.Port = 2379
	}
	if conf.Address == "" {
		conf.Address = publicAddress
	}
	etcdPeerPortStr := format.String(conf.PeerPort)
	etcdClientPortStr := format.String(conf.Port)

	etcdArgs := []string{
		"--name=node0",
		"--initial-advertise-peer-urls=http://" + conf.Address + ":" + etcdPeerPortStr,
		"--listen-peer-urls=http://" + conf.Address + ":" + etcdPeerPortStr,
		"--advertise-client-urls=http://" + conf.Address + ":" + etcdClientPortStr,
		"--listen-client-urls=http://" + conf.Address + ":" + etcdClientPortStr,
		"--initial-cluster=node0=http://" + conf.Address + ":" + etcdPeerPortStr,
		"--auto-compaction-retention=1",
		"--quota-backend-bytes=8589934592",
	}

	inContainer := conf.Image != ""
	var volumes []internalversion.Volume

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
	} else {
		etcdArgs = append(etcdArgs,
			"--data-dir="+conf.DataPath,
		)
	}

	return internalversion.Component{
		Name:    "etcd",
		Version: conf.Version.String(),
		Volumes: volumes,
		Command: []string{"etcd"},
		Args:    etcdArgs,
		Binary:  conf.Binary,
		Image:   conf.Image,
		WorkDir: conf.Workdir,
	}, nil
}
