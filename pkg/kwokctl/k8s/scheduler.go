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

package k8s

import (
	"fmt"

	"sigs.k8s.io/kwok/pkg/utils/file"
)

func CopySchedulerConfig(oldpath, newpath, kubeconfig string) error {
	err := file.Copy(oldpath, newpath)
	if err != nil {
		return err
	}

	err = file.Append(newpath, []byte(fmt.Sprintf(`
clientConnection:
  kubeconfig: %q
`, kubeconfig)))
	if err != nil {
		return err
	}

	return nil
}
