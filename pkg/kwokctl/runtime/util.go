/*
Copyright 2023 The Kubernetes Authors.

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
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// GetComponentPatches returns the patches for a component.
func GetComponentPatches(conf *internalversion.KwokctlConfiguration, componentName string) internalversion.ComponentPatches {
	componentPatches, _ := slices.Find(conf.ComponentsPatches, func(patch internalversion.ComponentPatches) bool {
		return patch.Name == componentName
	})
	return componentPatches
}

// ExpandVolumesHostPaths expands relative paths specified in volumes to absolute paths
func ExpandVolumesHostPaths(volumes []internalversion.Volume) ([]internalversion.Volume, error) {
	result := make([]internalversion.Volume, 0, len(volumes))
	for _, v := range volumes {
		hostPath, err := path.Expand(v.HostPath)
		if err != nil {
			return nil, err
		}
		v.HostPath = hostPath
		result = append(result, v)
	}
	return result, nil
}

// GetLogVolumes returns volumes for Logs and ClusterLogs resource.
func GetLogVolumes(ctx context.Context) ([]internalversion.Volume, error) {
	logs := config.FilterWithTypeFromContext[*internalversion.Logs](ctx)
	clusterLogs := config.FilterWithTypeFromContext[*internalversion.ClusterLogs](ctx)
	attaches := config.FilterWithTypeFromContext[*internalversion.Attach](ctx)
	clusterAttaches := config.FilterWithTypeFromContext[*internalversion.ClusterAttach](ctx)

	// Mount log dirs
	var mountDirs dirMountSet
	for _, log := range logs {
		for _, l := range log.Spec.Logs {
			mountDirs.add(l.LogsFile)
		}
	}

	for _, cl := range clusterLogs {
		for _, l := range cl.Spec.Logs {
			mountDirs.add(l.LogsFile)
		}
	}

	for _, attach := range attaches {
		for _, a := range attach.Spec.Attaches {
			mountDirs.add(a.LogsFile)
		}
	}

	for _, ca := range clusterAttaches {
		for _, a := range ca.Spec.Attaches {
			mountDirs.add(a.LogsFile)
		}
	}

	if mountDirs.err != nil {
		return nil, mountDirs.err
	}

	volumes := make([]internalversion.Volume, 0, mountDirs.size())
	i := 0
	for _, dir := range mountDirs.items() {
		dirPath := strings.TrimPrefix(dir, "/var/components/controller")
		volumes = append(volumes, internalversion.Volume{
			Name:      fmt.Sprintf("log-volume-%d", i),
			HostPath:  dirPath,
			MountPath: dirPath,
			PathType:  internalversion.HostPathDirectoryOrCreate,
			ReadOnly:  true,
		})
		i++
	}

	return volumes, nil
}

type dirMountSet struct {
	mounts map[string]struct{}
	err    error
}

func (m *dirMountSet) add(logsFile string) {
	if m.err != nil {
		return
	}
	if m.mounts == nil {
		m.mounts = make(map[string]struct{})
	}
	abs, err := path.Expand(logsFile)
	if err != nil {
		m.err = err
		return
	}
	m.mounts[filepath.Dir(abs)] = struct{}{}
}

func (m *dirMountSet) size() int {
	return len(m.mounts)
}

func (m *dirMountSet) items() []string {
	result := make([]string, 0, m.size())
	for mount := range m.mounts {
		result = append(result, mount)
	}
	return result
}
