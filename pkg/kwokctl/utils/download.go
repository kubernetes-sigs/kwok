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

package utils

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"sigs.k8s.io/kwok/pkg/log"
)

func DownloadWithCacheAndExtract(ctx context.Context, cacheDir, src, dest string, match string, mode fs.FileMode, quiet bool, clean bool) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	cacheTar, err := getCachePath(cacheDir, src)
	if err != nil {
		return err
	}
	cache := PathJoin(filepath.Dir(cacheTar), match)
	if _, err = os.Stat(cache); err != nil {
		cacheTar, err = getCacheOrDownload(ctx, cacheDir, src, 0644, quiet)
		if err != nil {
			return err
		}
		err = Untar(ctx, cacheTar, func(file string) (string, bool) {
			if filepath.Base(file) == match {
				return cache, true
			}
			return "", false
		})
		if err != nil {
			return err
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

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return err
	}

	// link the cache file to the dest file
	return os.Symlink(cache, dest)
}

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
	return os.Symlink(cache, dest)
}

func getCachePath(cacheDir, src string) (string, error) {
	u, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "https":
		return PathJoin(cacheDir, u.Scheme, u.Host, u.Path), nil
	default:
		return src, nil
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
		logger.Info("Download", "uri", src)

		cli := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return "", err
		}
		req = req.WithContext(ctx)
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

		err = os.MkdirAll(filepath.Dir(cache), 0755)
		if err != nil {
			return "", err
		}

		d, err := os.OpenFile(cache+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return "", err
		}

		var srcReader io.Reader = resp.Body
		if !quiet {
			pb := NewProgressBar()
			contentLength := resp.Header.Get("Content-Length")
			contentLengthInt, _ := strconv.Atoi(contentLength)
			counter := newCounterWriter(func(counter int) {
				pb.Update(counter, contentLengthInt)
				pb.Print()
			})
			srcReader = io.TeeReader(srcReader, counter)
		}

		_, err = io.Copy(d, srcReader)
		if err != nil {
			_ = d.Close()
			fmt.Println()
			return "", err
		}
		err = d.Close()
		if err != nil {
			logger.Error("Failed to close file", err)
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

type counterWriter struct {
	fun     func(counter int)
	counter int
}

func newCounterWriter(fun func(counter int)) *counterWriter {
	return &counterWriter{
		fun: fun,
	}
}
func (c *counterWriter) Write(b []byte) (int, error) {
	c.counter += len(b)
	c.fun(c.counter)
	return len(b), nil
}
