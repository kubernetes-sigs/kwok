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
	"sigs.k8s.io/kwok/pkg/utils/flock"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
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
	cache := utilspath.Join(utilspath.Dir(cacheTar), match)
	if _, err = os.Stat(cache); err != nil {
		cacheTar, err = getCacheOrDownload(ctx, cacheDir, src, 0644, quiet)
		if err != nil {
			return err
		}
		err = untar(ctx, cacheTar, func(file string) (string, bool) {
			if utilspath.Base(file) == match {
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

	err = MkdirAll(utilspath.Dir(dest))
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

	err = os.MkdirAll(filepath.Dir(dest), 0755)
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
		return utilspath.Join(cacheDir, u.Scheme, u.Host, u.Path), nil
	default:
		src, err = utilspath.Expand(src)
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
		return downloadHTTPWithCache(ctx, cache, src, u, mode, quiet)
	default:
		return src, nil
	}
}

func lockIncompleteFile(ctx context.Context, path string) (*os.File, error) {
	incompleteFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	var lastUpdate = time.Now()
	var offset int64
	for {
		err = flock.Lock(incompleteFile)
		if err == nil {
			return incompleteFile, nil
		}
		if !errors.Is(err, flock.ErrLocked) {
			_ = incompleteFile.Close()
			return nil, fmt.Errorf("failed to lock download cache: %w", err)
		}

		off, err := incompleteFile.Seek(0, io.SeekEnd)
		if err != nil {
			_ = incompleteFile.Close()
			return nil, err
		}
		if offset == off && time.Since(lastUpdate) > time.Minute {
			_ = incompleteFile.Close()
			return nil, fmt.Errorf("download is locked and not updated for more than 1 minute, please check the download process or remove the incomplete file: %s", path)
		}

		offset = off
		lastUpdate = time.Now()
		select {
		case <-ctx.Done():
			_ = incompleteFile.Close()
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func downloadHTTPWithCache(ctx context.Context, cache, src string, u *url.URL, mode fs.FileMode, quiet bool) (string, error) {
	err := os.MkdirAll(utilspath.Dir(cache), 0755)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cache); err == nil {
		return cache, nil
	}

	tmp := cache + ".incomplete"
	incompleteFile, err := lockIncompleteFile(ctx, tmp)
	if err != nil {
		return "", err
	}

	logger := log.FromContext(ctx).With("uri", src)
	closed := false
	defer func() {
		if closed {
			return
		}
		err := incompleteFile.Close()
		if err != nil {
			logger.Error("Failed to close incomplete file", "path", tmp, "err", err)
		}
	}()
	locked := true

	if _, err := os.Stat(cache); err == nil {
		err = flock.Unlock(incompleteFile)
		if err != nil {
			logger.Error("Failed to unlock download cache", "path", tmp, "err", err)
		}
		locked = false
		err = os.Remove(tmp)
		if err != nil {
			logger.Debug("Failed to remove incomplete file", "path", tmp, "err", err)
		}
		return cache, nil
	}
	defer func() {
		if !locked {
			return
		}
		err := flock.Unlock(incompleteFile)
		if err != nil {
			log.FromContext(ctx).Error("Failed to unlock download cache",
				"path", tmp,
				"err", err,
			)
		}
	}()

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

	offset, err := incompleteFile.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}
	if !quiet {
		transport = progressbar.NewTransportWithOffset(transport, uint64(offset))
	}
	cli := &http.Client{Transport: transport}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", version.DefaultUserAgent())
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logger.Error("Failed to close body of response", "err", err)
		}
	}()

	if offset == 0 {
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GET %s: %s", u.String(), resp.Status)
		}
	} else if resp.StatusCode != http.StatusPartialContent {
		return "", fmt.Errorf("GET Partial %s: %s", u.String(), resp.Status)
	}

	contentLength, err := io.Copy(incompleteFile, resp.Body)
	if err != nil {
		return "", err
	}
	if resp.ContentLength >= 0 && resp.ContentLength != contentLength {
		return "", fmt.Errorf("content length mismatch: %d != %d", resp.ContentLength, contentLength)
	}

	err = os.Chmod(tmp, mode)
	if err != nil {
		return "", err
	}
	err = flock.Unlock(incompleteFile)
	if err != nil {
		return "", err
	}
	locked = false
	err = incompleteFile.Close()
	if err != nil {
		return "", err
	}
	closed = true

	err = os.Rename(tmp, cache)
	if err != nil {
		if _, statErr := os.Stat(cache); statErr == nil {
			return cache, nil
		}
		return "", err
	}
	return cache, nil
}
