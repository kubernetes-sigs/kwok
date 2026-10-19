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
	"sigs.k8s.io/kwok/pkg/utils/version"
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

// The following runtime mode is classification of runtime for components.
const (
	RuntimeModeNative    = "native"
	RuntimeModeContainer = "container"
	RuntimeModeCluster   = "cluster"
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
	runtimeTypeMap = map[string]string{
		consts.RuntimeTypeBinary:      RuntimeModeNative,
		consts.RuntimeTypeDocker:      RuntimeModeContainer,
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
func GetRuntimeMode(runtime string) string {
	return runtimeTypeMap[runtime]
}

// readinessProbeTimeoutSeconds leaves room for the container runtimes, where each check is relayed through an exec.
const readinessProbeTimeoutSeconds = 5

// httpGetReadinessProbe returns a readiness probe that sends an HTTP GET request to the given path and port.
func httpGetReadinessProbe(path string, port uint32) *internalversion.ComponentProbe {
	return &internalversion.ComponentProbe{
		HTTPGet: &internalversion.ComponentHTTPGetAction{
			Path: path,
			Port: port,
		},
		TimeoutSeconds: readinessProbeTimeoutSeconds,
	}
}

// controlPlaneReadinessProbe returns the readiness probe of kube-controller-manager or kube-scheduler serving path on port,
// or nil if path cannot be reached without credentials, which the probe does not send.
// On the secure port, only the paths in --authorization-always-allow-paths skip the delegated authorization,
// which kwokctl sets for kube-controller-manager since Kubernetes 1.13, and always for kube-scheduler.
func controlPlaneReadinessProbe(secure bool, args []string, path string, port uint32) *internalversion.ComponentProbe {
	probe := httpGetReadinessProbe(path, port)
	if secure {
		flags := parseFlags(args)
		if _, ok := flags["authorization-always-allow-paths"]; !ok || checkAlwaysAllowPaths(flags, probe) != nil {
			return nil
		}
		probe.HTTPGet.Scheme = internalversion.URISchemeHTTPS
	}
	return probe
}

// addonReadinessProbe returns the readiness probe of an add-on that serves /readyz on port,
// which was verified for its release minimum.
// Earlier releases are not probed, so that an endpoint they may not serve does not keep them not ready.
func addonReadinessProbe(v, minimum version.Version, port uint32) *internalversion.ComponentProbe {
	if port == 0 || v.LT(minimum) {
		return nil
	}
	return httpGetReadinessProbe("/readyz", port)
}
