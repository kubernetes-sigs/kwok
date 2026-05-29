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

package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// EnsureManifest fetches a manifest and caches it in CacheDir.
func (c *Cluster) EnsureManifest(ctx context.Context, url string) ([]byte, error) {
	config, err := c.Config(ctx)
	if err != nil {
		return nil, err
	}

	cachePath := manifestCachePath(config.Options.CacheDir, url)

	if c.IsDryRun() {
		if isKustomizeManifestSource(url) {
			dryrun.PrintMessagef("# Build %s by kubectl kustomize to %s", url, cachePath)
			return nil, nil
		}
		dryrun.PrintMessagef("# Download %s to %s", url, cachePath)
		return nil, nil
	}

	if file.Exists(cachePath) {
		return os.ReadFile(cachePath)
	}

	var data []byte
	if isKustomizeManifestSource(url) {
		data, err = c.buildManifestByKustomize(ctx, url)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = fetchRemoteManifest(ctx, url)
		if err != nil {
			return nil, err
		}
	}

	err = os.MkdirAll(filepath.Dir(cachePath), 0750)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(cachePath, data, 0640)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func isKustomizeManifestSource(url string) bool {
	return !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")
}

func (c *Cluster) buildManifestByKustomize(ctx context.Context, source string) ([]byte, error) {
	var out bytes.Buffer
	err := c.Kubectl(utilsexec.WithAllWriteTo(ctx, &out), "kustomize", source)
	if err != nil {
		return nil, fmt.Errorf("failed to build remote manifest %q by kubectl kustomize: %w", source, err)
	}
	return out.Bytes(), nil
}

func fetchRemoteManifest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for manifest %q: %w", url, err)
	}
	req.Header.Set("User-Agent", version.DefaultUserAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote manifest %q: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch remote manifest %q: unexpected status %s", url, resp.Status)
	}
	if resp.ContentLength > 10*1024*1024 {
		return nil, fmt.Errorf("manifest %q is too large: %d bytes", url, resp.ContentLength)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote manifest %q: %w", url, err)
	}
	return data, nil
}

func manifestCachePath(cacheDir, url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(cacheDir, "manifest", hex.EncodeToString(sum[:])+".yaml")
}
