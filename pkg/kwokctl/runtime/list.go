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

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/sets"
)

// ListClusters returns the list of clusters in the directory
func ListClusters(ctx context.Context) ([]string, error) {
	workdir := config.ClustersDir
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

// GetUsedPorts returns the list of ports used by the clusters in the directory
func GetUsedPorts(ctx context.Context) (rets sets.Sets[uint32]) {
	workdir := config.ClustersDir

	logger := log.FromContext(ctx)
	rets = sets.Sets[uint32]{}
	entries, err := os.ReadDir(workdir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("Failed to read clusters directory", "path", workdir, "error", err)
		}
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			logger.Warn("Found non-directory entry in clusters directory, please remove it", "path", path.Join(workdir, name))
			continue
		}
		confPath := path.Join(workdir, name, consts.ConfigName)
		if !file.Exists(confPath) {
			logger.Warn("Found directory without a config file, please remove it", "path", path.Join(workdir, name))
			continue
		}

		confs, err := config.Load(ctx, confPath)
		if err != nil {
			logger.Warn("Failed to load config file", "path", confPath, "error", err)
			continue
		}
		kubectlConfs := config.FilterWithType[*internalversion.KwokctlConfiguration](confs)
		if len(kubectlConfs) == 0 {
			logger.Warn("Not found kwokctl config in cluster", "path", confPath)
			continue
		}

		kubectlConf := kubectlConfs[0]
		for _, component := range kubectlConf.Components {
			for _, port := range component.Ports {
				if port.HostPort != 0 {
					rets.Insert(port.HostPort)
				}
			}
		}
	}
	return rets
}
