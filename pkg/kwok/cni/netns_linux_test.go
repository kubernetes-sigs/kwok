package cni

import (
	"os"
	"path"
	"testing"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/stretchr/testify/assert"
)

// setupTestEnv creates a temporary directory for testing
func setupTestEnv() (string, error) {
	dir, err := os.MkdirTemp("", "cni-test")
	if err != nil {
		return "", err
	}
	return dir, nil
}

// teardownTestEnv cleans up the temporary directory
func teardownTestEnv(dir string) {
	os.RemoveAll(dir)
}

// TestGetNS tests the GetNS function
func TestGetNS(t *testing.T) {
	testDir, err := setupTestEnv()
	assert.NoError(t, err)
	defer teardownTestEnv(testDir)

	nsRunDir := getNsRunDir()
	testNsName := "cni-test"
	testNsPath := path.Join(nsRunDir, testNsName)

	// Create a mock namespace file
	f, err := os.Create(testNsPath)
	assert.NoError(t, err)
	f.Close()

	netns, err := GetNS("testns")
	assert.NoError(t, err)
	assert.NotNil(t, netns)

	// Clean up the mock namespace file
	err = os.Remove(testNsPath)
	assert.NoError(t, err)
}

// TestNewNS tests the NewNS function
func TestNewNS(t *testing.T) {
	testDir, err := setupTestEnv()
	assert.NoError(t, err)
	defer teardownTestEnv(testDir)

	nsRunDir := getNsRunDir()
	testNsName := "cni-kwok-testns"
	testNsPath := path.Join(nsRunDir, testNsName)

	// Clean up any pre-existing test files
	err = os.RemoveAll(testNsPath)
	assert.NoError(t, err)

	netns, err := NewNS("testns")
	assert.NoError(t, err)
	assert.NotNil(t, netns)

	// Verify that the namespace file was created
	_, err = os.Stat(testNsPath)
	assert.NoError(t, err)

	// Clean up the namespace file
	err = UnmountNS(netns)
	assert.NoError(t, err)
}

// TestUnmountNS tests the UnmountNS function
func TestUnmountNS(t *testing.T) {
	testDir, err := setupTestEnv()
	assert.NoError(t, err)
	defer teardownTestEnv(testDir)

	nsRunDir := getNsRunDir()
	testNsName := "cni-kwok-testns"
	testNsPath := path.Join(nsRunDir, testNsName)

	// Create a mock namespace file
	f, err := os.Create(testNsPath)
	assert.NoError(t, err)
	f.Close()

	netns, err := ns.GetNS(testNsPath)
	assert.NoError(t, err)

	err = UnmountNS(netns)
	assert.NoError(t, err)

	// Verify that the namespace file was removed
	_, err = os.Stat(testNsPath)
	assert.True(t, os.IsNotExist(err))
}
