/*
Copyright 2025 The Kubernetes Authors.

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

package config

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// FetchDefaultConfig fetches the default configuration file from the remote URL
// and saves it to the local file system if it doesn't already exist.
func FetchDefaultConfig(ctx context.Context) error {
	fullPath := path.Join(WorkDir, consts.ConfigName)
	if file.Exists(fullPath) {
		return nil
	}

	logger := log.FromContext(ctx)

	logger.Info("Fetch default config from remote",
		"url", consts.KwokctlDefaultConfigURL,
		"path", path.RelFromHome(fullPath),
	)

	err := file.MkdirAll(WorkDir)
	if err != nil {
		return err
	}

	f, err := file.Open(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, consts.KwokctlDefaultConfigURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", version.DefaultUserAgent())

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
