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

package e2e

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
	"sigs.k8s.io/kwok/test/e2e/helper"

	_ "embed"
)

//go:embed jsonpatch.yaml
var jsonpatchCase []byte

// CaseJsonpatch creates a feature that tests jsonpatch
func CaseJsonpatch(nodeName string, namespace string) *features.FeatureBuilder {
	const key = "kwok.x-k8s.io/test-jsonpatch-available"

	node := helper.NewNodeBuilder(nodeName).
		WithAnnotation(key, "status").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		WithAnnotation(key, "status").
		Build()

	return features.New("Jsonpatch Stage").
		Assess("test stage jsonpatch", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			err = v1alpha1.AddToScheme(client.GetScheme())
			if err != nil {
				t.Fatal(err)
			}

			logger := log.FromContext(ctx)

			decoder := yaml.NewDecoder(bytes.NewBuffer(jsonpatchCase))

			var ss []*v1alpha1.Stage
			for {
				var s *v1alpha1.Stage
				err := decoder.Decode(&s)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					t.Fatal(err)
				}
				err = client.Create(ctx, s)
				if err != nil {
					t.Fatal(err)
				}

				ss = append(ss, s)
			}

			err = wait.For(
				func(ctx context.Context) (done bool, err error) {
					var item v1alpha1.Stage
					if err = client.Get(ctx, ss[0].Name, ss[0].Namespace, &item); err != nil {
						logger.Error("failed to list stage", err)
						return false, nil
					}

					if item.Annotations[key] != "True" {
						logger.Info("waiting for stage to be ready")
						return false, nil
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
		}).
		Assess("create node", helper.CreateNode(node)).
		Assess("create pod", helper.CreatePod(pod0)).
		Assess("test node jsonpatch", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			logger := log.FromContext(ctx)

			err = wait.For(
				func(ctx context.Context) (done bool, err error) {
					var item corev1.Node
					if err = client.Get(ctx, node.Name, node.Namespace, &item); err != nil {
						logger.Error("failed to list node", err)
						return false, nil
					}

					if item.Status.Phase != corev1.NodeTerminated {
						logger.Info("waiting for node to be changed")
						return false, nil
					}

					return true, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(10*time.Second),
			)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("test pod jsonpatch", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			logger := log.FromContext(ctx)

			err = wait.For(
				func(ctx context.Context) (done bool, err error) {
					var item corev1.Pod
					if err = client.Get(ctx, pod0.Name, pod0.Namespace, &item); err != nil {
						logger.Error("failed to list pod", err)
						return false, nil
					}

					if item.Status.Phase != corev1.PodFailed {
						logger.Info("waiting for node to be changed")
						return false, nil
					}

					return true, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(10*time.Second),
			)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod", helper.DeletePod(pod0)).
		Assess("delete node", helper.DeleteNode(node))
}
