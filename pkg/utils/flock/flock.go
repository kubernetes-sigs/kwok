/*
Copyright 2026 The Kubernetes Authors.

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

package flock

import (
	"fmt"
	"os"
)

var (
	// ErrLocked indicates that another process holds the advisory file lock.
	ErrLocked = fmt.Errorf("file is already locked")
)

// Lock acquires an advisory lock on the file.
func Lock(f *os.File) error {
	return lock(f)
}

// Unlock releases the advisory lock on the file.
func Unlock(f *os.File) error {
	return unlock(f)
}
