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
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
)

var (
	// ErrBrokenLinks is returned when there are broken links.
	ErrBrokenLinks = fmt.Errorf("broken links dependency detected")
)

// GroupByLinks groups stages by links.
func GroupByLinks(components []internalversion.Component) ([][]internalversion.Component, error) {
	had := sets.NewString()
	next := slices.Clone(components)
	groups := [][]internalversion.Component{}

	for len(next) != 0 {
		current := next
		next = next[:0]
		group := []internalversion.Component{}

		for _, component := range current {
			if len(component.Links) != 0 && !had.HasAll(component.Links...) {
				next = append(next, component)
				continue
			}
			group = append(group, component)
		}
		if len(group) == 0 {
			if len(next) != 0 {
				next := utilsslices.Map(next, func(component internalversion.Component) string {
					return component.Name
				})
				return nil, fmt.Errorf("%w: %v", ErrBrokenLinks, next)
			}
		} else {
			added := utilsslices.Map(group, func(component internalversion.Component) string {
				return component.Name
			})
			had.Insert(added...)
			groups = append(groups, group)
		}
	}
	return groups, nil
}

type RuntimeMode string

const (
	RuntimeModeNative    RuntimeMode = "native"
	RuntimeModeContainer RuntimeMode = "container"
	RuntimeModeCluster   RuntimeMode = "cluster"
)

const (
	kubeconfigPath   = "/etc/kubernetes/kubeconfig.yaml"
	pkiCACertPath    = "/etc/kubernetes/pki/ca.crt"
	pkiAdminCertPath = "/etc/kubernetes/pki/admin.crt"
	pkiAdminKeyPath  = "/etc/kubernetes/pki/admin.key"
	metricsPath      = "/metrics"
	schemeHTTPS      = "https"
)

var (
	runtimeTypeMap = map[string]RuntimeMode{
		consts.RuntimeTypeBinary:      RuntimeModeNative,
		consts.RuntimeTypeDocker:      RuntimeModeContainer,
		consts.RuntimeTypeDockerHost:  RuntimeModeContainer,
		consts.RuntimeTypePodman:      RuntimeModeContainer,
		consts.RuntimeTypeNerdctl:     RuntimeModeContainer,
		consts.RuntimeTypeLima:        RuntimeModeContainer,
		consts.RuntimeTypeFinch:       RuntimeModeContainer,
		consts.RuntimeTypeKind:        RuntimeModeCluster,
		consts.RuntimeTypeKindPodman:  RuntimeModeCluster,
		consts.RuntimeTypeKindNerdctl: RuntimeModeCluster,
		consts.RuntimeTypeKindLima:    RuntimeModeCluster,
		consts.RuntimeTypeKindFinch:   RuntimeModeCluster,
	}
)

// GetRuntimeMode returns the mode of runtime.
func GetRuntimeMode(runtime string) RuntimeMode {
	return runtimeTypeMap[runtime]
}

type RuntimeNetwork string

const (
	RuntimeNetworkHost    RuntimeNetwork = "host"
	RuntimeNetworkBridge  RuntimeNetwork = "bridge"
	RuntimeNetworkCluster RuntimeNetwork = "cluster"
)

var (
	runtimeNetwork = map[string]RuntimeNetwork{
		consts.RuntimeTypeBinary:      RuntimeNetworkHost,
		consts.RuntimeTypeDocker:      RuntimeNetworkBridge,
		consts.RuntimeTypeDockerHost:  RuntimeNetworkHost,
		consts.RuntimeTypePodman:      RuntimeNetworkBridge,
		consts.RuntimeTypeNerdctl:     RuntimeNetworkBridge,
		consts.RuntimeTypeLima:        RuntimeNetworkBridge,
		consts.RuntimeTypeFinch:       RuntimeNetworkBridge,
		consts.RuntimeTypeKind:        RuntimeNetworkCluster,
		consts.RuntimeTypeKindPodman:  RuntimeNetworkCluster,
		consts.RuntimeTypeKindNerdctl: RuntimeNetworkCluster,
		consts.RuntimeTypeKindLima:    RuntimeNetworkCluster,
		consts.RuntimeTypeKindFinch:   RuntimeNetworkCluster,
	}
)

// GetRuntimeNetwork returns the network mode of runtime.
func GetRuntimeNetwork(runtime string) RuntimeNetwork {
	return runtimeNetwork[runtime]
}
