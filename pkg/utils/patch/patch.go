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

package patch

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// StrategicMerge is a wrapper around the StrategicMergePatch function that
// takes in a generic type and returns a generic type.
func StrategicMerge[T any](original, patch T) (result T, err error) {
	patchData, err := json.Marshal(patch)
	if err != nil {
		return result, err
	}

	return StrategicMergePatch(original, patchData)
}

// StrategicMergePatch is a wrapper around the strategicpatch.StrategicMergePatch
// function that takes in a generic type and returns a generic type.
func StrategicMergePatch[T any](original T, patchData []byte) (result T, err error) {
	ori, err := json.Marshal(original)
	if err != nil {
		return result, err
	}

	sum, err := strategicpatch.StrategicMergePatch(ori, patchData, original)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(sum, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
