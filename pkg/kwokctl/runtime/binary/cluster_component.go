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

package binary

import (
	"context"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// AddComponent adds the component in to cluster
func (c *Cluster) AddComponent(ctx context.Context, name string, args ...string) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}
	_, ok := slices.Find(conf.Components, func(component internalversion.Component) bool {
		return component.Name == name
	})
	if ok {
		return fmt.Errorf("component %s is already exists", name)
	}

	kcp := config.FilterWithTypeFromContext[*internalversion.KwokctlComponent](ctx)
	renderer := gotpl.NewRenderer(gotpl.FuncMap{
		"ClusterName": c.Name,
		"Workdir":     c.Workdir,
		"Runtime": func() string {
			return c.Runtime(ctx)
		},
		"Mode": func() string {
			return c.Mode(ctx)
		},
		"Address": func() string {
			return c.ComponentAddress(ctx, name)
		},
		"PkiDir": func() string {
			return path.Join(c.Workdir(), runtime.PkiName)
		},
		"Kubeconfig": func() string {
			return c.GetWorkdirPath(runtime.InHostKubeconfigName)
		},
		"Config": func() *internalversion.KwokctlConfiguration {
			return conf
		},
	})

	krc, ok := slices.Find(kcp, func(krc *internalversion.KwokctlComponent) bool {
		return krc.Name == name
	})
	if !ok {
		return fmt.Errorf("component %s is not exists", name)
	}

	param, err := expression.NewParameters(ctx, krc.Parameters, args)
	if err != nil {
		return err
	}

	componentData, err := renderer.ToJSON(krc.Template, param)
	if err != nil {
		return err
	}
	var component internalversion.Component
	err = json.Unmarshal(componentData, &component)
	if err != nil {
		return err
	}
	component.Name = name

	binaryPath, err := c.EnsureBinary(ctx, component.Name, component.Binary)
	if err != nil {
		return err
	}

	component.Binary = binaryPath

	conf.Components = append(conf.Components, component)

	return c.SetConfig(ctx, conf)
}
