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

package k8s

import (
	"fmt"
	"testing"
)

func TestRawData(t *testing.T) {
	for i, data := range rawData {
		if err := data.Verification(); err != nil {
			t.Error(data, err)
		}

		if data.Until >= 0 && i+1 < len(rawData) {
			nextData := rawData[i+1]
			if data.Name == nextData.Name {
				if data.Until+1 != nextData.Since {
					t.Error(data, fmt.Errorf("invalid until: %d + 1 != next since: %d", data.Until, data.Since))
				}
			}
		}
	}
}
