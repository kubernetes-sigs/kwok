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

package exec

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
)

func TestKillProcess(t *testing.T) {
	// Start a new process (e.g., sleep for 10 seconds)
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start test process: %v", err)
	}
	pid := cmd.Process.Pid

	// Ensure the process is running
	if !IsRunning(pid) {
		t.Fatalf("expected process %d to be running", pid)
	}

	// Kill the process
	err := KillProcess(pid)
	if err != nil {
		t.Fatalf("failed to kill process %d: %v", pid, err)
	}
	err = cmd.Wait()
	if err != nil {
		// Check if the error indicates the process was killed
		var exiterr *exec.ExitError
		if errors.As(err, &exiterr) {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && status.Signal() == syscall.SIGKILL {
					t.Logf("process %d was killed as expected", pid)
				} else {
					t.Fatalf("process %d exited with status %v, expected to be killed", pid, status)
				}
			} else {
				t.Fatalf("failed to get exit status for process %d: %v", pid, err)
			}
		}
	} else {
		t.Fatalf("process %d exited normally, expected to be killed", pid)
	}

	// Ensure the process is no longer running
	if IsRunning(pid) {
		t.Fatalf("expected process %d to be stopped", pid)
	}
}
