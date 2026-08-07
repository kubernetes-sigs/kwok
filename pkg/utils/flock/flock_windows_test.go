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
	"os"
	"path/filepath"
	"testing"
)

func TestLockDoesNotPreventWritingFileContents(t *testing.T) {
	file, err := OpenFile(filepath.Join(t.TempDir(), "test.lock"))
	if err != nil {
		t.Fatalf("failed to open lock file: %v", err)
	}
	defer file.Close()

	if err := TryLock(file); err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer Unlock(file)

	if _, err := file.Write([]byte("contents")); err != nil {
		t.Fatalf("failed to write locked file: %v", err)
	}
}

func TestOpenFileAllowsRenameWhileAnotherProcessWaits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	first, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open first lock file: %v", err)
	}
	defer first.Close()
	if err := TryLock(first); err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}
	defer Unlock(first)

	second, err := OpenFile(path)
	if err != nil {
		t.Fatalf("failed to open second lock file: %v", err)
	}
	defer second.Close()

	if err := os.Rename(path, path+".complete"); err != nil {
		t.Fatalf("failed to rename file while another process waits: %v", err)
	}
}
