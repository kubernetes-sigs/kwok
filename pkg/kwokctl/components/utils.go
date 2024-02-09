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
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// ErrBrokenLinks is returned when there are broken links.
var ErrBrokenLinks = fmt.Errorf("broken links dependency detected")

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
				next := slices.Map(next, func(component internalversion.Component) string {
					return component.Name
				})
				return nil, fmt.Errorf("%w: %v", ErrBrokenLinks, next)
			}
		} else {
			added := slices.Map(group, func(component internalversion.Component) string {
				return component.Name
			})
			had.Insert(added...)
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func AddExtraArgs(args []string, extraArgs []string) []string {
	for _, arg := range extraArgs {
		splits := strings.SplitN(arg, "=", 3)
		if len(splits) != 3 {
			continue
		}
		if splits[0] != consts.ComponentEtcd {
			continue
		}
		args = append(args, fmt.Sprintf("--%s=%s", splits[1], splits[2]))
	}
	return args
}

// The following runtime mode is classification of runtime for components.
const (
	RuntimeModeNative    = "native"
	RuntimeModeContainer = "container"
	RuntimeModeCluster   = "cluster"
)

var runtimeTypeMap = map[string]string{
	consts.RuntimeTypeKind:       RuntimeModeCluster,
	consts.RuntimeTypeKindPodman: RuntimeModeCluster,
	consts.RuntimeTypeDocker:     RuntimeModeContainer,
	consts.RuntimeTypeNerdctl:    RuntimeModeContainer,
	consts.RuntimeTypePodman:     RuntimeModeContainer,
	consts.RuntimeTypeBinary:     RuntimeModeNative,
}

// GetRuntimeMode returns the mode of runtime.
func GetRuntimeMode(runtime string) string {
	return runtimeTypeMap[runtime]
}
