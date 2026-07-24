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
	"os"
	"path/filepath"
	"testing"
)

func TestLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	first, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to open first lock file: %v", err)
	}
	defer first.Close()
	if err := Lock(first); err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}

	second, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to open second lock file: %v", err)
	}
	defer second.Close()
	if err := Lock(second); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected lock to be held, got %v", err)
	}

	if err := Unlock(first); err != nil {
		t.Fatalf("failed to unlock first lock: %v", err)
	}
	if err := Lock(second); err != nil {
		t.Fatalf("failed to acquire second lock: %v", err)
	}
	if err := Unlock(second); err != nil {
		t.Fatalf("failed to unlock second lock: %v", err)
	}
}
