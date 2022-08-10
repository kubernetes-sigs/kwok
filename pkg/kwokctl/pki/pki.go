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

package pki

import (
	_ "embed"
	"os"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
)

var (
	//go:embed testdata/ca.crt
	CACrt []byte
	//go:embed testdata/admin.key
	AdminKey []byte
	//go:embed testdata/admin.crt
	AdminCrt []byte
)

// DumpPki generates a pki directory.
func DumpPki(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(dir, "ca.crt"), CACrt, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(dir, "admin.key"), AdminKey, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(utils.PathJoin(dir, "admin.crt"), AdminCrt, 0644)
	if err != nil {
		return err
	}
	return nil
}
