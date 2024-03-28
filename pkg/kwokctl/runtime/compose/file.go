/*
Copyright 2024 The Kubernetes Authors.

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

package compose

import (
	"context"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

func (c *Cluster) setupFiles(_ context.Context, env *env) (err error) {
	r := gotpl.NewRenderer(nil)
	pathDir := c.GetWorkdirPath("files")
	for index, component := range env.kwokctlConfig.Components {
		for i, file := range component.Files {
			var data []byte
			if file.Data != "" {
				data = []byte(file.Data)
			} else if file.Template != "" {
				d, err := r.ToText(file.Template, env.kwokctlConfig)
				if err != nil {
					return err
				}
				data = d
			}

			filePath := path.Join(pathDir, component.Name, file.Path)

			err = c.MkdirAll(path.Dir(filePath))
			if err != nil {
				return err
			}
			wc, err := c.OpenFile(filePath)
			if err != nil {
				return err
			}
			_, err = wc.Write(data)
			if err != nil {
				return err
			}

			err = wc.Close()
			if err != nil {
				return err
			}

			v := internalversion.Volume{
				Name:      "file-" + format.String(i),
				MountPath: file.Path,
				HostPath:  filePath,
			}

			env.kwokctlConfig.Components[index].Volumes = append(env.kwokctlConfig.Components[index].Volumes, v)
		}
	}
	return nil
}
