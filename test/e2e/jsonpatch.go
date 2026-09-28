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
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/klient/k8s"
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

const (
	// jsonpatchKey is the annotation the Stages in jsonpatch.yaml select on.
	jsonpatchKey = "kwok.x-k8s.io/test-jsonpatch-available"
	// jsonpatchTouchKey is an annotation that is rewritten to trigger a new
	// event for an object without changing anything the Stages select on.
	jsonpatchTouchKey = "kwok.x-k8s.io/test-jsonpatch-touch"
	// jsonpatchTimeout bounds each wait for a Stage patch to be applied.
	jsonpatchTimeout = 2 * time.Minute
)

// CaseJsonpatch creates a feature that tests jsonpatch.
//
// kwok matches Stages against an object only when it receives an event for
// that object, and nothing re-evaluates existing objects when the set of
// Stages changes. The Stages created here can therefore become effective
// after the last event of their target has already been processed, in which
// case the target is never patched on its own. Every wait in this feature
// touches its target between polls so that kwok evaluates it again against
// the Stages it currently has.
func CaseJsonpatch(nodeName string, namespace string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(nodeName).
		WithAnnotation(jsonpatchKey, "status").
		Build()
	pod0 := helper.NewPodBuilder("pod0").
		WithNamespace(namespace).
		WithNodeName(nodeName).
		WithAnnotation(jsonpatchKey, "status").
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

			// The first Stage selects itself, so it is both a rule and a
			// target and can be processed as a target before it is visible
			// as a rule.
			err = waitForJsonpatch(ctx, client, ss[0], func(ctx context.Context) (bool, error) {
				var item v1alpha1.Stage
				if err := client.Get(ctx, ss[0].Name, ss[0].Namespace, &item); err != nil {
					logger.Error("failed to get stage",
						"err", err,
					)
					return false, nil
				}

				if item.Annotations[jsonpatchKey] != "True" {
					logger.Info("waiting for stage to be patched")
					return false, nil
				}

				return true, nil
			})
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

			err = waitForJsonpatch(ctx, client, node, func(ctx context.Context) (bool, error) {
				var item corev1.Node
				if err := client.Get(ctx, node.Name, node.Namespace, &item); err != nil {
					logger.Error("failed to get node",
						"err", err,
					)
					return false, nil
				}

				if item.Status.Phase != corev1.NodeTerminated {
					logger.Info("waiting for node to be patched")
					return false, nil
				}

				return true, nil
			})
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

			err = waitForJsonpatch(ctx, client, pod0, func(ctx context.Context) (bool, error) {
				var item corev1.Pod
				if err := client.Get(ctx, pod0.Name, pod0.Namespace, &item); err != nil {
					logger.Error("failed to get pod",
						"err", err,
					)
					return false, nil
				}

				if item.Status.Phase != corev1.PodFailed {
					logger.Info("waiting for pod to be patched")
					return false, nil
				}

				return true, nil
			})
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("delete pod", helper.DeletePod(pod0)).
		Assess("delete node", helper.DeleteNode(node))
}

// waitForJsonpatch waits until check reports that obj has been patched by its
// Stage. Between polls it touches obj so that kwok receives a new event for it
// and matches it again against the Stages it currently has, which makes the
// wait independent of whether the Stage was already loaded when obj was first
// processed.
func waitForJsonpatch(ctx context.Context, client *resources.Resources, obj k8s.Object, check func(ctx context.Context) (bool, error)) error {
	logger := log.FromContext(ctx)

	return wait.For(
		func(ctx context.Context) (bool, error) {
			done, err := check(ctx)
			if err != nil || done {
				return done, err
			}

			if err := touchObject(ctx, client, obj); err != nil {
				logger.Error("failed to touch object",
					"err", err,
				)
			}
			return false, nil
		},
		wait.WithContext(ctx),
		wait.WithTimeout(jsonpatchTimeout),
	)
}

// touchObject rewrites an annotation on obj that no Stage selects on, so that
// the apiserver emits a new event for obj.
func touchObject(ctx context.Context, client *resources.Resources, obj k8s.Object) error {
	data, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				jsonpatchTouchKey: strconv.FormatInt(time.Now().UnixNano(), 10),
			},
		},
	})
	if err != nil {
		return err
	}

	return client.Patch(ctx, obj, k8s.Patch{
		PatchType: types.MergePatchType,
		Data:      data,
	})
}
