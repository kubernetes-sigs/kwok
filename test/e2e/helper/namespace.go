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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// CreateNamespace creates a namespace
func CreateNamespace(name string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		client, err := cfg.NewClient()
		if err != nil {
			return ctx, fmt.Errorf("create namespace func: %w", err)
		}

		resource := client.Resources()

		err = resource.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
		if err != nil {
			return ctx, fmt.Errorf("create namespace func: %w", err)
		}
		cfg.WithNamespace(name) // set env config default namespace

		err = waitForServiceAccountReady(ctx, resource, "default", name)
		if err != nil {
			return ctx, fmt.Errorf("wait for service account ready: %w", err)
		}
		return ctx, nil
	}
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(name string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		client, err := cfg.NewClient()
		if err != nil {
			return ctx, fmt.Errorf("delete namespace func: %w", err)
		}

		resource := client.Resources()
		err = resource.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
		if err != nil {
			return ctx, fmt.Errorf("delete namespace func: %w", err)
		}

		return ctx, nil
	}
}
