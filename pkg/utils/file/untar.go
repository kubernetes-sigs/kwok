/*
Copyright 2022 The Kubernetes Authors.

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

package file

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kwok/pkg/log"
)

// untar untars the given tarball to the given destination.
func untar(ctx context.Context, src string, filter func(file string) (string, bool)) error {
	if strings.HasSuffix(src, ".tar.gz") {
		return untargz(ctx, src, filter)
	} else if strings.HasSuffix(src, ".zip") {
		return unzip(ctx, src, filter)
	}
	return fmt.Errorf("unsupported archive format: %s", src)
}

func unzip(ctx context.Context, src string, filter func(file string) (string, bool)) error {
	logger := log.FromContext(ctx)
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		err = r.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
		}
	}()

	for _, f := range r.File {
		fi := f.FileInfo()

		name := f.Name
		if fi.IsDir() {
			continue
		}

		err = func() error {
			name, ok := filter(name)
			if !ok {
				return nil
			}

			err = os.MkdirAll(filepath.Dir(name), 0750)
			if err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() {
				err = rc.Close()
				if err != nil {
					logger.Error("Failed to close file of tar", err)
				}
			}()

			outFile, err := os.OpenFile(filepath.Clean(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
			if err != nil {
				return err
			}
			defer func() {
				err = outFile.Close()
				if err != nil {
					logger.Error("Failed to close file", err)
				}
			}()

			return copy(outFile, rc)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func untargz(ctx context.Context, src string, filter func(file string) (string, bool)) error {
	logger := log.FromContext(ctx)
	r, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() {
		err = r.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
		}
	}()

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		err = gzr.Close()
		if err != nil {
			logger.Error("Failed to close gzip reader", err)
		}
	}()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		name := hdr.Name
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		err = func() error {
			name, ok := filter(name)
			if !ok {
				return nil
			}

			err = os.MkdirAll(filepath.Dir(name), 0755)
			if err != nil {
				return err
			}
			outFile, err := os.OpenFile(filepath.Clean(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			defer func() {
				err = outFile.Close()
				if err != nil {
					logger.Error("Failed to close file", err)
				}
			}()

			return copy(outFile, tr)
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func copy(dst io.Writer, src io.Reader) error {
	for {
		_, err := io.CopyN(dst, src, 1024)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
	}
	return nil
}