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

package utils

import (
	"os"
	"path/filepath"
)

func CreateFile(name string, perm os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(name), 0755)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func CopyFile(oldpath, newpath string) error {
	err := os.MkdirAll(filepath.Dir(newpath), 0755)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(oldpath)
	if err != nil {
		return err
	}

	err = os.WriteFile(newpath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
