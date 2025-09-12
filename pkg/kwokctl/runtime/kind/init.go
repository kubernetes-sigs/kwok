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

package kind

import (
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
)

func init() {
	runtime.DefaultRegistry.RegisterDeprecated(consts.RuntimeTypeKind, NewDockerCluster)
	runtime.DefaultRegistry.RegisterDeprecated(consts.RuntimeTypeKindPodman, NewPodmanCluster)
	runtime.DefaultRegistry.RegisterDeprecated(consts.RuntimeTypeKindNerdctl, NewNerdctlCluster)
	runtime.DefaultRegistry.RegisterDeprecated(consts.RuntimeTypeKindLima, NewLimaCluster)
	runtime.DefaultRegistry.RegisterDeprecated(consts.RuntimeTypeKindFinch, NewFinchCluster)
}
