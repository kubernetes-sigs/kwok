//go:build windows

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
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openFile(path string) (*os.File, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		if err := windows.CloseHandle(handle); err != nil {
			return nil, fmt.Errorf("failed to close file handle: %w", err)
		}
		return nil, fmt.Errorf("failed to create file from handle")
	}
	return file, nil
}

func tryLock(f *os.File) error {
	// Lock a byte far beyond the file contents. Locking the start of the file
	// prevents Windows from writing downloaded data through this file handle.
	// Keeping the lock outside the data also lets waiters inspect the current
	// download size.
	overlapped := windows.Overlapped{OffsetHigh: 0x80000000}
	err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &overlapped)
	if err == nil {
		return nil
	}
	if !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return err
	}
	return ErrLocked
}

func unlock(f *os.File) error {
	overlapped := windows.Overlapped{OffsetHigh: 0x80000000}
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlapped)
}
