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

func GetLogVolumes(ctx context.Context) []internalversion.Volume {
	logs := config.FilterWithTypeFromContext[*internalversion.Logs](ctx)
	clusterLogs := config.FilterWithTypeFromContext[*internalversion.ClusterLogs](ctx)

	// Mount log dirs
	mountDirs := make(map[string]struct{})
	for _, log := range logs {
		for _, l := range log.Spec.Logs {
			mountDirs[filepath.Dir(l.LogsFile)] = struct{}{}
		}
	}

	for _, cl := range clusterLogs {
		for _, l := range cl.Spec.Logs {
			mountDirs[filepath.Dir(l.LogsFile)] = struct{}{}
		}
	}

	volumes := make([]internalversion.Volume, 0, len(mountDirs))
	i := 0
	for dir := range mountDirs {
		volumes = append(volumes, internalversion.Volume{
			Name:      fmt.Sprintf("log-volume-%d", i),
			HostPath:  dir,
			MountPath: dir,
			PathType:  internalversion.HostPathDirectoryOrCreate,
			ReadOnly:  true,
		})
		i++
	}

	return volumes
}
