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
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support"
)

// CreateCluster creates a cluster
func CreateCluster(p support.E2EClusterProvider, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		kubecfg, err := p.Create(ctx, args...)
		if err != nil {
			return ctx, err
		}

		cfg.WithKubeconfigFile(kubecfg)

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		err = wait.For(conditions.New(r).ResourceListN(&corev1.ServiceAccountList{}, 1))
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

// DestroyCluster destroys a cluster
func DestroyCluster(p support.E2EClusterProvider) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		err := p.Destroy(ctx)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

// ExportLogs exports logs from a cluster
func ExportLogs(p support.E2EClusterProvider, dest string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		err := p.ExportLogs(ctx, dest)
		if err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}
