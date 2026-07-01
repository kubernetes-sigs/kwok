/*
Copyright 2023 The Kubernetes Authors.

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

package runtime

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/pki"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
)

// DownloadWithCache downloads the src file to the dest file.
// The src can be:
//   - A URL to download directly (e.g. https://example.com/kube-scheduler)
//   - A URL#filename to extract from an archive (e.g. https://example.com/etcd.tar.gz#etcd)
//   - A URL#subdirectory to build from source (e.g. https://example.com/v0.34.7.tar.gz#cmd/controller)
func (c *Cluster) DownloadWithCache(ctx context.Context, cacheDir, src, dest string, mode fs.FileMode, quiet bool) error {
	if s := strings.SplitN(src, "#", 2); len(s) == 2 {
		if strings.Contains(s[1], "/") {
			// URL#subdirectory pattern: build from source
			return c.buildBinaryFromSource(ctx, dest, s[0], s[1])
		}
		if c.IsDryRun() {
			dryrun.PrintMessagef("# Download %s and extract %s to %s", s[0], s[1], dest)
			return nil
		}
		return file.DownloadWithCacheAndExtract(ctx, cacheDir, s[0], dest, s[1], mode, quiet, true)
	}

	if c.IsDryRun() {
		dryrun.PrintMessagef("# Download %s to %s", src, dest)
		return nil
	}
	return file.DownloadWithCache(ctx, cacheDir, src, dest, mode, quiet)
}

// buildBinaryFromSource builds a binary from source by downloading and extracting
// a source archive, then running go build.
//   - binaryPath: the destination path for the built binary
//   - archiveURL: URL to the source archive (.tar.gz)
//   - subdir: subdirectory within the archive to build (e.g. "cmd/controller")
func (c *Cluster) buildBinaryFromSource(ctx context.Context, binaryPath, archiveURL, subdir string) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := config.Options

	archiveName := strings.TrimSuffix(filepath.Base(archiveURL), ".tar.gz")

	// Source directory after extraction
	srcDir := filepath.Join(conf.CacheDir, "src", archiveName)

	cacheBinaryName := archiveName + "-" + strings.ReplaceAll(subdir, "/", "-")
	cacheBinaryPath := filepath.Join(conf.CacheDir, "gobin", cacheBinaryName)

	// Check if the binary is already cached
	if _, err := os.Stat(cacheBinaryPath); err == nil {
		err = os.MkdirAll(filepath.Dir(binaryPath), 0755)
		if err != nil {
			return err
		}
		return os.Symlink(cacheBinaryPath, binaryPath)
	}

	if c.IsDryRun() {
		dryrun.PrintMessagef("# Download %s and build %s from source to %s", archiveURL, subdir, binaryPath)
		return nil
	}

	// Download the source archive
	archiveCachePath, err := file.GetCachePath(conf.CacheDir, archiveURL)
	if err != nil {
		return err
	}
	err = c.DownloadWithCache(ctx, conf.CacheDir, archiveURL, archiveCachePath, 0644, conf.QuietPull)
	if err != nil {
		return err
	}

	// Extract the archive to the source directory, stripping the top-level directory
	err = file.UntarTo(ctx, archiveCachePath, srcDir, 1)
	if err != nil {
		return err
	}

	// Ensure the gobin directory exists
	err = c.MkdirAll(filepath.Dir(cacheBinaryPath))
	if err != nil {
		return err
	}

	// Build the binary from the extracted source with dynamic linking
	err = utilsexec.Exec(utilsexec.WithDir(ctx, srcDir), "go", "build", "-o", cacheBinaryPath, "./"+subdir)
	if err != nil {
		return err
	}

	// Symlink from binaryPath to the cached binary
	err = os.MkdirAll(filepath.Dir(binaryPath), 0755)
	if err != nil {
		return err
	}
	err = os.Symlink(cacheBinaryPath, binaryPath)
	if err != nil {
		return err
	}

	return nil
}

// GeneratePki generates the pki for kwokctl
func (c *Cluster) GeneratePki(pkiPath string, sans ...string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("# Generate PKI to %s", pkiPath)
		return nil
	}

	return pki.GeneratePki(pkiPath, sans...)
}

// CreateFile creates a file.
func (c *Cluster) CreateFile(name string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("touch %s", name)
		return nil
	}

	return file.Create(name)
}

// CopyFile copies a file from src to dst.
func (c *Cluster) CopyFile(oldpath, newpath string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("cp %s %s", oldpath, newpath)
		return nil
	}

	return file.Copy(oldpath, newpath)
}

// RenameFile renames a file.
func (c *Cluster) RenameFile(oldpath, newpath string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("mv %s %s", oldpath, newpath)
		return nil
	}

	return file.Rename(oldpath, newpath)
}

// AppendToFile appends content to a file.
func (c *Cluster) AppendToFile(name string, content []byte) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("cat <<EOF >>%s\n%s\nEOF", name, string(content))
		return nil
	}

	return file.Append(name, content)
}

// Remove removes a file.
func (c *Cluster) Remove(name string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("rm %s", name)
		return nil
	}

	return file.Remove(name)
}

// RemoveAll removes a directory and all its contents.
func (c *Cluster) RemoveAll(name string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("rm -rf %s", name)
		return nil
	}

	return file.RemoveAll(name)
}

// OpenFile opens/creates a file for writing.
func (c *Cluster) OpenFile(name string) (io.WriteCloser, error) {
	if c.IsDryRun() {
		return dryrun.NewCatToFileWriter(name), nil
	}

	return file.Open(name)
}

// WriteFile writes content to a file.
func (c *Cluster) WriteFile(name string, content []byte) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("cat <<EOF >%s\n%s\nEOF", name, string(content))
		return nil
	}

	return file.Write(name, content)
}

// MkdirAll creates a directory.
func (c *Cluster) MkdirAll(name string) error {
	if c.IsDryRun() {
		dryrun.PrintMessagef("mkdir -p %s", name)
		return nil
	}

	return file.MkdirAll(name)
}

// EnsureBinary ensures the binary exists.
func (c *Cluster) EnsureBinary(ctx context.Context, name, binary string) (string, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return "", err
	}
	conf := config.Options

	binaryPath := c.GetBinPath(name + conf.BinSuffix)

	// Check if the binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	err = c.DownloadWithCache(ctx, conf.CacheDir, binary, binaryPath, 0755, conf.QuietPull)
	if err != nil {
		return "", err
	}

	return binaryPath, nil
}
