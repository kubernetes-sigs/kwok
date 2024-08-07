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

// Package stage provides a stage tester.
package stage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// TestingStages tests the stages against the target object
func TestingStages(ctx context.Context, target any, stages []*internalversion.Stage) (any, error) {
	testTarget, ok := target.(Obj)
	if !ok {
		return nil, fmt.Errorf("expected target to be an object, got %T", target)
	}

	gvk := testTarget.GetObjectKind().GroupVersionKind()

	want := internalversion.StageResourceRef{
		APIGroup: gvk.GroupVersion().String(),
		Kind:     gvk.Kind,
	}

	meta := map[string]any{
		"apiGroup": want.APIGroup,
		"kind":     want.Kind,
		"name":     testTarget.GetName(),
	}
	if ns := testTarget.GetNamespace(); ns != "" {
		meta["namespace"] = ns
	}

	stages = slices.Filter(stages, func(stage *internalversion.Stage) bool {
		return stage.Spec.ResourceRef == want
	})

	lc, err := lifecycle.NewLifecycle(stages)
	if err != nil {
		return nil, err
	}

	lcstages, err := lc.ListAllPossible(ctx, testTarget.GetLabels(), testTarget.GetAnnotations(), testTarget)
	if err != nil {
		return nil, err
	}

	out := []any{}
	for _, stage := range lcstages {
		o, err := testingStage(ctx, testTarget, stage)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}

	meta["stages"] = out

	return meta, nil
}

var now = time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)

func testingStage(ctx context.Context, testTarget Obj, stage *lifecycle.Stage) (any, error) {
	meta := map[string]any{
		"stage": stage.Name(),
	}

	delay, ok := stage.DelayRangePossible(ctx, testTarget, now)
	if ok {
		meta["delay"] = delay
	}

	weight, ok := stage.Weight(ctx, testTarget)
	if ok {
		meta["weight"] = weight
	}

	next := stage.Next()

	if next == nil {
		meta["next"] = "nil"
		return meta, nil
	}

	out := make([]any, 0)

	patch, err := next.Finalizers(testTarget.GetFinalizers())
	if err != nil {
		return nil, err
	}

	if patch != nil {
		out = append(out, formatPatch(patch))
	}

	if next.Delete() {
		out = append(out, map[string]string{
			"kind": "delete",
		})
		meta["next"] = out
		return meta, nil
	}

	fm := gotpl.FuncMap{}
	funcNames := []string{
		// For node and pod
		"NodeIP",

		// For node
		"NodeName",
		"NodePort",

		// For pod
		"PodIP",
		"NodeIPWith",
		"PodIPWith",

		// Override built-in
		"Now",
		"now",
		"Version",
	}
	for _, name := range funcNames {
		fm[name] = wrapFunction(name)
	}

	renderer := gotpl.NewRenderer(fm)

	patches, err := next.Patches(testTarget, renderer)
	if err != nil {
		return nil, err
	}

	for _, patch := range patches {
		out = append(out, formatPatch(patch))
	}

	if stage.ImmediateNextStage() {
		out = append(out, map[string]string{
			"kind": "immediate",
		})
	}

	meta["next"] = out
	return meta, nil
}

func wrapFunction(name string) func(args ...any) any {
	return func(args ...any) any {
		if len(args) == 0 {
			return fmt.Sprintf("<%s>", name)
		}

		return fmt.Sprintf("<%s(%s)>", name,
			strings.Join(
				slices.Map(args,
					func(arg any) string {
						a := fmt.Sprintf("%#v", arg)
						if a == "" {
							return `""`
						}
						return a
					},
				),
				", ",
			),
		)
	}
}

func formatPatch(patch *lifecycle.Patch) any {
	out := map[string]any{
		"kind": "patch",
	}
	out["type"] = patch.Type
	if patch.Subresource != "" {
		out["subresource"] = patch.Subresource
	}
	out["data"] = json.RawMessage(patch.Data)

	if patch.Impersonation != nil {
		out["impersonation"] = patch.Impersonation.Username
	}

	return out
}

type Obj interface {
	metav1.Object
	runtime.Object
}
