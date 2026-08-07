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

// TryLock attempts to acquire an exclusive advisory lock on the open file f
// without blocking. It returns ErrLocked if another process holds the lock.
func TryLock(f *os.File) error {
	return tryLock(f)
}

// Unlock releases the advisory lock on the file.
func Unlock(f *os.File) error {
	return unlock(f)
}

// OpenFile opens path for reading and writing, creating it if it does not exist.
// On Windows, the returned file permits another process to rename it while it is
// open.
func OpenFile(path string) (*os.File, error) {
	return openFile(path)
}
