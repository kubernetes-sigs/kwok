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

package compose

import (
	"strings"

	"github.com/compose-spec/compose-go/types"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func convertToCompose(name string, hostIP string, cs []internalversion.Component) *types.Config {
	svcs := convertComponentsToComposeServices(name, hostIP, cs)
	return &types.Config{
		Extensions: map[string]interface{}{
			"version": "3",
		},
		Services: svcs,
		Networks: map[string]types.NetworkConfig{
			"default": {
				Name: name,
			},
		},
	}
}

func convertComponentsToComposeServices(prefix string, hostIP string, cs []internalversion.Component) (svcs types.Services) {
	svcs = make(types.Services, len(cs))
	for i, c := range cs {
		svcs[i] = convertComponentToComposeService(prefix, hostIP, c)
	}
	return svcs
}

func convertComponentToComposeService(prefix string, hostIP string, cs internalversion.Component) (svc types.ServiceConfig) {
	svc.Name = cs.Name
	svc.ContainerName = prefix + "-" + cs.Name
	svc.Image = cs.Image
	svc.Links = cs.Links
	svc.Entrypoint = cs.Command
	if svc.Entrypoint == nil {
		svc.Entrypoint = []string{}
	}
	svc.Command = cs.Args
	svc.Restart = "always"
	svc.Ports = make([]types.ServicePortConfig, len(cs.Ports))
	for i, p := range cs.Ports {
		svc.Ports[i] = types.ServicePortConfig{
			Mode:      "ingress",
			HostIP:    hostIP,
			Target:    p.Port,
			Published: format.String(p.HostPort),
			Protocol:  strings.ToLower(string(p.Protocol)),
		}
	}
	svc.Environment = make(map[string]*string, len(cs.Envs))
	for i, e := range cs.Envs {
		svc.Environment[e.Name] = &cs.Envs[i].Value
	}
	svc.Volumes = make([]types.ServiceVolumeConfig, len(cs.Volumes))
	for i, v := range cs.Volumes {
		svc.Volumes[i] = types.ServiceVolumeConfig{
			Type:     types.VolumeTypeBind,
			Source:   v.HostPath,
			Target:   v.MountPath,
			ReadOnly: v.ReadOnly,
		}
	}
	return svc
}
