/*
Copyright 2023 The Kubernetes Authors.

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

package runtime

import (
	"fmt"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// GetComponentExtraArgs returns the extra args for a components patches.
func GetComponentExtraArgs(conf *internalversion.KwokctlConfiguration, componentName string) []string {
	componentPatches, ok := slices.Find(conf.ComponentsPatches, func(patch internalversion.ComponentPatches) bool {
		return patch.Name == componentName
	})
	if !ok {
		return []string{}
	}
	args := slices.Map(componentPatches.ExtraArgs, func(arg internalversion.ExtraArgs) string {
		return fmt.Sprintf("--%s=%s", arg.Key, arg.Value)
	})
	return args
}
