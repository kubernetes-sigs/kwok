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
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/wzshiming/httpseek"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/progressbar"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// DownloadWithCacheAndExtract downloads the src file to the dest file, and extract it to the dest directory.
func DownloadWithCacheAndExtract(ctx context.Context, cacheDir, src, dest string, match string, mode fs.FileMode, quiet bool, clean bool) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	cacheTar, err := getCachePath(cacheDir, src)
	if err != nil {
		return err
	}
	cache := path.Join(path.Dir(cacheTar), match)
	if _, err = os.Stat(cache); err != nil {
		cacheTar, err = getCacheOrDownload(ctx, cacheDir, src, 0644, quiet)
		if err != nil {
			return err
		}
		err = untar(ctx, cacheTar, func(file string) (string, bool) {
			if path.Base(file) == match {
				return cache, true
			}
			return "", false
		})
		if err != nil {
			return fmt.Errorf("failed to untar %s: %w", cacheTar, err)
		}
		if clean {
			err = os.Remove(cacheTar)
			if err != nil {
				return err
			}
		}
		err = os.Chmod(cache, mode)
		if err != nil {
			return err
		}
	}

	err = MkdirAll(path.Dir(dest))
	if err != nil {
		return err
	}

	// link the cache file to the dest file
	err = os.Symlink(cache, dest)
	if err != nil {
		return err
	}
	return nil
}

// DownloadWithCache downloads the src file to the dest file.
func DownloadWithCache(ctx context.Context, cacheDir, src, dest string, mode fs.FileMode, quiet bool) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	cache, err := getCacheOrDownload(ctx, cacheDir, src, mode, quiet)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(dest), 0750)
	if err != nil {
		return err
	}

	// link the cache file to the dest file
	err = os.Symlink(cache, dest)
	if err != nil {
		return err
	}
	return nil
}

func getCachePath(cacheDir, src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "https":
		return path.Join(cacheDir, u.Scheme, u.Host, u.Path), nil
	default:
		src, err = path.Expand(src)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(src); err != nil {
			return "", err
		}
		return src, err
	}
}

func getCacheOrDownload(ctx context.Context, cacheDir, src string, mode fs.FileMode, quiet bool) (string, error) {
	cache, err := getCachePath(cacheDir, src)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cache); err == nil {
		return cache, nil
	}

	u, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "https":

		logger := log.FromContext(ctx)
		logger = logger.With(
			"uri", src,
		)
		logger.Info("Download")

		var transport = http.DefaultTransport
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

		cli := &http.Client{
			Transport: transport,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", version.DefaultUserAgent())
		resp, err := cli.Do(req)
		if err != nil {
			return "", err
		}

		defer func() {
			err = resp.Body.Close()
			if err != nil {
				logger.Error("Failed to close body of response", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("%s: %s", u.String(), resp.Status)
		}

		err = os.MkdirAll(path.Dir(cache), 0750)
		if err != nil {
			return "", err
		}

		d, err := os.OpenFile(cache+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return "", err
		}

		contentLength, err := io.Copy(d, resp.Body)
		if err != nil {
			_ = d.Close()
			fmt.Println()
			return "", err
		}
		err = d.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
		}
		if resp.ContentLength != contentLength {
			return "", fmt.Errorf("content length mismatch: %d != %d", resp.ContentLength, contentLength)
		}

		err = os.Rename(cache+".tmp", cache)
		if err != nil {
			return "", err
		}
		return cache, nil
	default:
		return src, nil
	}
}
