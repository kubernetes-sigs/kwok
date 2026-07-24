/*
Copyright 2026 The Kubernetes Authors.

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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetCacheOrDownloadConcurrent(t *testing.T) {
	t.Parallel()

	const body = "concurrent download cache content"
	var requests atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()

	const workers = 8
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	pathCh := make(chan string, workers)

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()

			path, err := getCacheOrDownload(context.Background(), cacheDir, srv.URL+"/archive.tgz", 0644, true)
			if err != nil {
				errCh <- err
				return
			}
			pathCh <- path
		}()
	}

	wg.Wait()
	close(errCh)
	close(pathCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("getCacheOrDownload returned error: %v", err)
		}
	}

	var cachePath string
	for path := range pathCh {
		if cachePath == "" {
			cachePath = path
			continue
		}
		if path != cachePath {
			t.Fatalf("expected a shared cache path, got %q and %q", cachePath, path)
		}
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("failed to read cache file %q: %v", cachePath, err)
	}
	if string(data) != body {
		t.Fatalf("unexpected cache contents: got %q, want %q", string(data), body)
	}

	if _, err := os.Stat(cachePath + ".incomplete"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete download to be renamed, got err=%v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("expected exactly one cold-start download, got %d", got)
	}
}

func TestGetCacheOrDownloadResumesIncompleteFile(t *testing.T) {
	t.Parallel()

	const body = "partial download content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "archive.tgz", time.Now(), strings.NewReader(body))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	cachePath, err := getCachePath(cacheDir, srv.URL+"/archive.tgz")
	if err != nil {
		t.Fatalf("failed to build cache path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}
	if err := os.WriteFile(cachePath+".incomplete", []byte(body[:8]), 0644); err != nil {
		t.Fatalf("failed to create incomplete download: %v", err)
	}

	incomplete, err := os.ReadFile(cachePath + ".incomplete")
	if err != nil {
		t.Fatalf("failed to read incomplete download: %v", err)
	}
	if got, want := string(incomplete), body[:8]; got != want {
		t.Fatalf("unexpected incomplete download: got %q, want %q", got, want)
	}

	path, err := getCacheOrDownload(context.Background(), cacheDir, srv.URL+"/archive.tgz", 0644, true)
	if err != nil {
		t.Fatalf("failed to resume download: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read resumed download: %v", err)
	}
	if got := string(data); got != body {
		t.Fatalf("unexpected resumed download: got %q, want %q", got, body)
	}
	if _, err := os.Stat(cachePath + ".incomplete"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete download to be renamed, got err=%v", err)
	}
}
