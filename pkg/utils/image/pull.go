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
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/wzshiming/httpseek"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/progressbar"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Format is the image archive format used when exporting images.
type Format string

const (
	// FormatDocker exports images as Docker image tarball format.
	FormatDocker Format = "docker"
	// FormatOCI exports images as OCI-compatible tar archive.
	FormatOCI Format = "oci"
)

// Pull pulls an image from a registry.
func Pull(ctx context.Context, cacheDir, src, dest string, quiet bool, format Format) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"image", src,
	)
	logger.Info("Pull")

	if format == "" {
		format = FormatDocker
	}

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

	switch format {
	case FormatOCI:
		err = saveOCIArchive(img, dest)
		if err != nil {
			return fmt.Errorf("saving oci archive %s: %w", dest, err)
		}
	default:
		err = crane.Save(img, src, dest)
		if err != nil {
			return fmt.Errorf("saving tarball %s: %w", dest, err)
		}
	}

	return nil
}

func saveOCIArchive(img containerregistryv1.Image, dest string) error {
	layoutDir, err := os.MkdirTemp(filepath.Dir(dest), "oci-layout-")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(layoutDir)
	}()

	err = crane.SaveOCI(img, layoutDir)
	if err != nil {
		return err
	}

	return tarDir(layoutDir, dest)
}

func tarDir(srcDir, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	tw := tar.NewWriter(out)
	defer func() {
		_ = tw.Close()
	}()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() {
			header.Name += "/"
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			header.Linkname = link
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()

		_, err = io.Copy(tw, f)
		return err
	})
}
