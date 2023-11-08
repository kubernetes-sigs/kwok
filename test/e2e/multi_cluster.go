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

package e2e

import (
	"context"
	"fmt"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kwok"

	"sigs.k8s.io/kwok/test/e2e/helper"
)

// CaseMultiCluster is a test case for multi-cluster
func CaseMultiCluster(kwokctlPath, logsDir string, replicas int, args ...string) *features.FeatureBuilder {
	f := features.New("Multi Cluster")
	name := envconf.RandomName("test", 16)
	for i := 0; i < replicas; i++ {
		n := fmt.Sprintf("%s-%d", name, i)
		k := kwok.NewCluster(n).
			WithPath(kwokctlPath)
		f = f.Assess("create cluster "+n, func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctx, err := helper.CreateCluster(k, args...)(ctx, cfg)
			if err != nil {
				t.Fatalf("create cluster %s failed: %v", n, err)
			}
			return ctx
		}).Assess("export cluster logs "+n, func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			ctx, err := helper.ExportLogs(k, logsDir)(ctx, config)
			if err != nil {
				t.Fatalf("export cluster logs %s failed: %v", n, err)
			}
			return ctx
		}).Teardown(func(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
			ctx, err := helper.DestroyCluster(k)(ctx, config)
			if err != nil {
				t.Fatalf("destroy cluster %s failed: %v", n, err)
			}
			return ctx
		})
	}
	return f
}
