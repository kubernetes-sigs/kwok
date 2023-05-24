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

package runtime

import (
	"context"
	"os"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

// ListClusters returns the list of clusters in the directory
func ListClusters(ctx context.Context, workdir string) ([]string, error) {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	logger := log.FromContext(ctx)
	ret := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			logger.Warn("Found non-directory entry in clusters directory, please remove it", "path", path.Join(workdir, name))
			continue
		}
		if !file.Exists(path.Join(workdir, name, consts.ConfigName)) {
			logger.Warn("Found directory without a config file, please remove it", "path", path.Join(workdir, name))
			continue
		}

		ret = append(ret, name)
	}
	return ret, nil
}
