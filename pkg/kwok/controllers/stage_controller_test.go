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

package controllers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic/fake"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/informer"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func TestStageController(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PersistentVolume{})
	client := fake.NewSimpleDynamicClient(scheme,
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-0",
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumePending,
			},
		},
	)

	gvr := corev1.SchemeGroupVersion.WithResource("persistentvolumes")

	lc, _ := lifecycle.NewLifecycle([]*internalversion.Stage{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pv-available",
			},
			Spec: internalversion.StageSpec{
				Selector: &internalversion.StageSelector{
					MatchExpressions: []internalversion.SelectorExpression{
						{
							SelectorJQ: &internalversion.SelectorJQ{
								Key:      ".status.phase",
								Operator: "NotIn",
								Values:   []string{"Available"},
							},
						},
					},
				},
				Next: internalversion.StageNext{
					Patches: []internalversion.StagePatch{
						{
							Template:    `phase: Available`,
							Subresource: "status",
							Root:        "status",
						},
					},
				},
			},
		},
	}, nil)
	patchMeta, _ := strategicpatch.NewPatchMetaFromStruct(corev1.PersistentVolume{})
	controller, err := NewStageController(StageControllerConfig{
		PlayStageParallelism: 1,
		GVR:                  gvr,
		DynamicClient:        client,
		Schema:               patchMeta,
		Lifecycle:            resources.NewStaticGetter(lc),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.LevelDebug))
	ctx, cancel := context.WithTimeout(ctx, 1000*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	resourceCh := make(chan informer.Event[*unstructured.Unstructured], 1)

	podsInformer := informer.NewInformer[*unstructured.Unstructured, *unstructured.UnstructuredList](client.Resource(gvr))
	err = podsInformer.Watch(ctx, informer.Option{}, resourceCh)
	if err != nil {
		t.Fatal(fmt.Errorf("watch resource error: %w", err))
	}

	err = controller.Start(ctx, resourceCh)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	data, err := client.Resource(gvr).Get(ctx, "pv-0", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got, err := yaml.Convert[*corev1.PersistentVolume](data)
	if err != nil {
		t.Fatal(err)
	}

	if got.Status.Phase != corev1.VolumeAvailable {
		t.Fatalf("expected phase %q, got %q", corev1.VolumeAvailable, got.Status.Phase)
	}
}
