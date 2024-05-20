package exec

import (
	"os/exec"
	"testing"
	"time"
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

	// Ensure the process is no longer running
	if IsRunning(pid) {
		t.Fatalf("expected process %d to be stopped", pid)
	}
}

func TestIsRunning(t *testing.T) {
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
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("failed to kill test process: %v", err)
	}

	// Wait for the process to exit
	if _, err := cmd.Process.Wait(); err != nil {
		t.Fatalf("failed to wait for process to exit: %v", err)
	}

	// Ensure the process is no longer running
	time.Sleep(1 * time.Second) // Give the system some time to update the process state
	if IsRunning(pid) {
		t.Fatalf("expected process %d to be stopped", pid)
	}
}
