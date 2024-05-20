package exec

import (
	"os"
	"testing"
)

func TestLookPath(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		expectErr bool
	}{
		{"ExistingExecutable", "ls", false},                   // A common Unix command
		{"NonExistingExecutable", "nonexistentcommand", true}, // A command that does not exist
		{"EmptyString", "", true},                             // An empty string
		{"PathToExecutable", "/usr/bin/cat", false},           // Full path to an existing executable
		{"PathToNonExecutableFile", "/etc/hosts", true},       // Full path to a non-executable file
		{"Directory", "/usr/bin", true},                       // A directory
		{"SpecialCharacters", "weird$cmd&name", true},         // Command na              // A relative path to an existing executable
		{"PathToNonExecutableDirectory", "/usr/lib", true},    // Full path to a non-executable directory
		// Full path to an executable with space
		{"ExecutableInCurrentDirectory", "./test_executable", false}, // Executable in the current directory
	}

	// Create a test executable file in the current directory
	testFile := "test_executable"
	if err := os.WriteFile(testFile, []byte("#!/bin/sh\necho 'Hello, World!'"), 0755); err != nil {
		t.Fatalf("failed to create test executable file: %v", err)
	}
	defer os.Remove(testFile) // Clean up

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LookPath(tt.file)
			if (err != nil) != tt.expectErr {
				t.Fatalf("LookPath(%s) error = %v, expectErr %v", tt.file, err, tt.expectErr)
			}
		})
	}
}
