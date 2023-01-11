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

package cni

import (
	"context"
	"fmt"

	gocni "github.com/containerd/go-cni"
)

func SupportedCNI() bool {
	return true
}

func Setup(ctx context.Context, id, name, namespace string) (ip []string, err error) {
	netns, err := NewNS(id)
	if err != nil {
		return nil, err
	}

	// Initializes library
	l, err := gocni.New(gocni.WithDefaultConf)
	if err != nil {
		return nil, err
	}

	// Setup network
	result, err := l.Setup(ctx, id, netns.Path(), getOpts(id, name, namespace))
	if err != nil {
		return nil, err
	}

	ips := []string{}
	for _, iface := range result.Interfaces {
		for _, conf := range iface.IPConfigs {
			if conf.IP != nil {
				ips = append(ips, conf.IP.String())
			}
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to set the network, no ip address was obtained: netns: %s, id: %s, name:%s, namespace:%s", netns, id, name, namespace)
	}
	return ips, nil
}

func Remove(ctx context.Context, id, name, namespace string) (err error) {
	netns, err := GetNS(id)
	if err != nil {
		return err
	}

	// Initializes library
	l, err := gocni.New(gocni.WithDefaultConf)
	if err != nil {
		return err
	}

	// Remove network
	err = l.Remove(ctx, id, netns.Path(), getOpts(id, name, namespace))
	if err != nil {
		return err
	}

	err = UnmountNS(netns)
	if err != nil {
		return err
	}
	return nil
}

func getOpts(id, name, namespace string) gocni.NamespaceOpts {
	// Setup network for namespace.
	labels := map[string]string{
		"K8S_POD_NAMESPACE":          namespace,
		"K8S_POD_NAME":               name,
		"K8S_POD_INFRA_CONTAINER_ID": id,
		"K8S_POD_UID":                id,

		// Plugin tolerates all Args embedded by unknown labels, like
		// K8S_POD_NAMESPACE/NAME/INFRA_CONTAINER_ID...
		"IgnoreUnknown": "1",
	}
	return gocni.WithLabels(labels)
}
