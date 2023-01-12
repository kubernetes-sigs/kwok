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

package file

import (
	"io"
	"os"
	"path/filepath"
)

// Create creates a file at path with the given content.
func Create(name string, perm os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(name), 0750)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	return nil
}

// Copy copies a file from src to dst.
func Copy(oldpath, newpath string) error {
	err := os.MkdirAll(filepath.Dir(newpath), 0750)
	if err != nil {
		return err
	}

	oldFile, err := os.OpenFile(oldpath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer func() {
		_ = oldFile.Close()
	}()

	fi, err := oldFile.Stat()
	if err != nil {
		return err
	}

	newFile, err := os.OpenFile(newpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer func() {
		_ = newFile.Close()
	}()

	_, err = io.Copy(newFile, oldFile)
	if err != nil {
		return err
	}
	return nil
}

// Append appends content to a file.
func Append(name string, content []byte) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = file.Write(content)
	if err != nil {
		return err
	}
	return nil
}

// Exists checks if a file exists.
func Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
