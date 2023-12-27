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
	"time"

	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/yaml"

	_ "embed"
)

//go:embed stage.yaml
var stageCase []byte

// CaseStage creates a feature that tests node creation and deletion
func CaseStage() *features.FeatureBuilder {
	return features.New("Resource Stage").
		Assess("test stage", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			err = v1alpha1.AddToScheme(client.GetScheme())
			if err != nil {
				t.Fatal(err)
			}

			var s *v1alpha1.Stage
			err = yaml.Unmarshal(stageCase, &s)
			if err != nil {
				t.Fatal(err)
			}

			err = client.Create(ctx, s)
			if err != nil {
				t.Fatal(err)
			}

			logger := log.FromContext(ctx)

			err = wait.For(
				func(ctx context.Context) (done bool, err error) {
					var item v1alpha1.Stage
					if err = client.Get(ctx, s.Name, s.Namespace, &item); err != nil {
						logger.Error("failed to list stage", err)
						return false, nil
					}

					conds := item.Status.Conditions
					if len(conds) == 0 {
						logger.Info("waiting for stage to be ready")
						return false, nil
					}

					if conds[0].Type != "Available" {
						return false, fmt.Errorf("stage is not available: %v", conds[0])
					}

					return true, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(600*time.Second),
			)
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		})
}
