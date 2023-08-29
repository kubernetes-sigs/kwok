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

package helper

import (
	"bytes"
	"context"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// CreateByKustomize applies kustomize to the cluster
func CreateByKustomize(kustomizeDir string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		initYAML, err := buildKustomizeToYaml(kustomizeDir)
		if err != nil {
			return nil, err
		}

		err = decoder.DecodeEach(ctx, bytes.NewReader(initYAML), decoder.CreateHandler(r))
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// DeleteByKustomize deletes kustomize from the cluster
func DeleteByKustomize(kustomizeDir string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		initYAML, err := buildKustomizeToYaml(kustomizeDir)
		if err != nil {
			return nil, err
		}

		err = decoder.DecodeEach(ctx, bytes.NewReader(initYAML), decoder.DeleteHandler(r))
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// buildKustomizeToYaml builds kustomize to yaml
func buildKustomizeToYaml(dir string) ([]byte, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	fs := filesys.MakeFsOnDisk()

	objs, err := k.Run(fs, dir)
	if err != nil {
		return nil, err
	}

	yaml, err := objs.AsYaml()
	if err != nil {
		return nil, err
	}
	return yaml, err
}
