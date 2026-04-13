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

package image

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/wzshiming/httpseek"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/progressbar"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Pull pulls an image from a registry.
func Pull(ctx context.Context, cacheDir, src, dest string, quiet bool) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"image", src,
	)
	logger.Info("Pull")

	var transport = remote.DefaultTransport
	transport = httpseek.NewMustReaderTransport(transport, func(req *http.Request, retry int, err error) error {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if retry > 10 {
			return err
		}
		logger.Warn("Retry after 1s",
			"err", err,
			"retry", retry,
		)
		time.Sleep(time.Second)
		return nil
	})

	if !quiet {
		transport = progressbar.NewTransport(transport)
	}

	o := crane.GetOptions(
		crane.WithContext(ctx),
		crane.WithUserAgent(version.DefaultUserAgent()),
		crane.WithTransport(transport),
		crane.WithPlatform(&containerregistryv1.Platform{
			OS:           "linux",
			Architecture: runtime.GOARCH,
		}),
	)

	ref, err := name.ParseReference(src, o.Name...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %w", src, err)
	}

	rmt, err := remote.Get(ref, o.Remote...)
	if err != nil {
		return err
	}

	img, err := rmt.Image()
	if err != nil {
		return err
	}
	if cacheDir != "" {
		img = cache.Image(img, cache.NewFilesystemCache(cacheDir))
	}

	err = saveOCIArchive(img, src, dest)
	if err != nil {
		return fmt.Errorf("saving OCI archive %s: %w", dest, err)
	}

	return nil
}

// saveOCIArchive saves an image as an OCI archive (tarball of OCI Image Layout).
func saveOCIArchive(img containerregistryv1.Image, ref, dest string) error {
	tmpDir, err := os.MkdirTemp("", "oci-layout-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p, err := layout.Write(tmpDir, empty.Index)
	if err != nil {
		return fmt.Errorf("writing OCI layout: %w", err)
	}

	err = p.AppendImage(img, layout.WithAnnotations(map[string]string{
		"org.opencontainers.image.ref.name": ref,
	}))
	if err != nil {
		return fmt.Errorf("appending image to layout: %w", err)
	}

	return tarDir(tmpDir, dest)
}

// tarDir creates a tar archive from a directory.
func tarDir(srcDir, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", dest, err)
	}

	tw := tar.NewWriter(f)

	walkErr := filepath.Walk(srcDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, file)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		return copyFileToTar(tw, file)
	})
	if walkErr != nil {
		_ = tw.Close()
		_ = f.Close()
		return walkErr
	}

	if err := tw.Close(); err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

// copyFileToTar copies the contents of a file into a tar writer.
func copyFileToTar(tw *tar.Writer, file string) error {
	data, err := os.Open(file)
	if err != nil {
		return err
	}
	defer func() { _ = data.Close() }()

	_, err = io.Copy(tw, data)
	return err
}
