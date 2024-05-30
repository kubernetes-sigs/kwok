/*
Copyright 2024 The Kubernetes Authors.

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

package signals

import (
	"syscall"
	"testing"
	"time"
)

// TestSetupSignalContext verifies that the SetupSignalContext function correctly handles shutdown signals.
func TestSetupSignalContext(t *testing.T) {
	// Create a context using SetupSignalContext
	ctx := SetupSignalContext()

	// Send a shutdown signal
	shutdownHandler <- syscall.SIGTERM

	// Check if the context is canceled
	select {
	case <-ctx.Done():
		// Context is canceled as expected
	case <-time.After(1 * time.Second):
		t.Errorf("Context was not canceled after shutdown signal")
	}

	// Reset global variables to allow repeated tests
	onlyOneSignalHandler = make(chan struct{})
	shutdownHandler = nil
}

// TestSetupSignalContextTwice verifies that calling SetupSignalContext twice panics.
func TestSetupSignalContextTwice(t *testing.T) {
	// First call should work
	_ = SetupSignalContext()

	// Second call should panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when calling SetupSignalContext twice, but did not panic")
		}
	}()

	_ = SetupSignalContext()
}
