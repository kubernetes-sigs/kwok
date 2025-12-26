/*
Copyright 2025 The Kubernetes Authors.

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

// Package stage_test provides tests for the stage cr.
package stage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

var (
	ctx            = context.Background()
	pwd            = os.Getenv("PWD")
	updateTestdata = false
)

func init() {
	flag.BoolVar(&updateTestdata, "update-testdata", false, "update all of testdata")
}

func TestStageCRs(t *testing.T) {
	inputFiles, err := filepath.Glob(filepath.Join(pwd, "*", "*", "testdata", "*.input.yaml"))
	if err != nil {
		t.Fatalf("failed to list input files: %v", err)
	}

	for _, file := range inputFiles {
		stageFiles, err := filepath.Glob(filepath.Join(file, "..", "..", "*.yaml"))
		if err != nil {
			t.Fatalf("failed to list stage files: %v", err)
		}

		stageFiles = slices.Filter(stageFiles, func(f string) bool {
			return filepath.Base(f) != "kustomization.yaml"
		})

		result, err := Testing(ctx, file, stageFiles)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		outFile := file[:len(file)-len(".input.yaml")] + ".output.yaml"

		testName, _ := filepath.Rel(pwd, file)
		testName = strings.TrimSuffix(testName, ".input.yaml")
		testName = strings.Replace(testName, "/testdata/", "/", 1)
		t.Run(testName, func(t *testing.T) {
			if updateTestdata {
				f, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
				if err != nil {
					t.Fatalf("failed to open input file: %v", err)
				}

				err = yaml.NewEncoder(f).Encode(result)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				buf := bytes.NewBuffer(nil)
				err = yaml.NewEncoder(buf).Encode(result)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				expected, err := os.ReadFile(outFile)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}

				if diff := cmp.Diff(string(expected), buf.String()); diff != "" {
					t.Errorf("output mismatch (-expected +got):\n%s", diff)
				}
			}
		})
	}
}

func BenchmarkStageCRs(b *testing.B) {
	inputFiles, err := filepath.Glob(filepath.Join(pwd, "*", "*", "testdata", "*.input.yaml"))
	if err != nil {
		b.Fatalf("failed to list input files: %v", err)
	}

	for _, file := range inputFiles {
		stageFiles, err := filepath.Glob(filepath.Join(file, "..", "..", "*.yaml"))
		if err != nil {
			b.Fatalf("failed to list stage files: %v", err)
		}

		stageFiles = slices.Filter(stageFiles, func(f string) bool {
			return filepath.Base(f) != "kustomization.yaml"
		})

		call, err := Benchmarking(ctx, file, stageFiles)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}

		testName, _ := filepath.Rel(pwd, file)
		testName = strings.TrimSuffix(testName, ".input.yaml")
		testName = strings.Replace(testName, "/testdata/", "/", 1)
		b.Run(testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := call()
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Testing tests the stages against the resource file.
func Testing(ctx context.Context, resourceFile string, stageFile []string) (any, error) {
	rio, err := config.LoadUnstructured(resourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load resource file %s: %w", resourceFile, err)
	}
	if len(rio) != 1 {
		return nil, fmt.Errorf("expected exactly one resource in %s, got %d", resourceFile, len(rio))
	}

	sio, err := config.Load(ctx, stageFile...)
	if err != nil {
		return nil, fmt.Errorf("failed to load stage files %v: %w", stageFile, err)
	}

	ue := config.FilterWithoutType[*internalversion.Stage](sio)
	if len(ue) != 0 {
		return nil, fmt.Errorf("expected only stage, got %d non-stage", len(ue))
	}

	stages := config.FilterWithType[*internalversion.Stage](sio)
	if len(stages) == 0 {
		return nil, fmt.Errorf("expected at least one stage, got 0")
	}

	testTarget, ok := rio[0].(obj)
	if !ok {
		return nil, fmt.Errorf("expected target to be an object, got %T", rio[0])
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
		return nil, fmt.Errorf("failed to create lifecycle: %w", err)
	}

	event := &lifecycle.Event{
		Labels:      testTarget.GetLabels(),
		Annotations: testTarget.GetAnnotations(),
		Data:        testTarget,
	}

	lcstages, err := lc.ListAllPossible(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to list all possible stages: %w", err)
	}

	out := []any{}
	for _, stage := range lcstages {
		o, err := testingStage(ctx, testTarget, event, stage)
		if err != nil {
			return nil, fmt.Errorf("failed to test stage %s: %w", stage.Name(), err)
		}
		out = append(out, o)
	}

	meta["stages"] = out

	return meta, nil
}

// Benchmarking benchmarks the stages against the resource file.
func Benchmarking(ctx context.Context, resourceFile string, stageFile []string) (func() (*lifecycle.Stage, error), error) {
	rio, err := config.LoadUnstructured(resourceFile)
	if err != nil {
		return nil, err
	}
	if len(rio) != 1 {
		return nil, fmt.Errorf("expected exactly one resource in %s, got %d", resourceFile, len(rio))
	}

	sio, err := config.Load(ctx, stageFile...)
	if err != nil {
		return nil, err
	}

	ue := config.FilterWithoutType[*internalversion.Stage](sio)
	if len(ue) != 0 {
		return nil, fmt.Errorf("expected only stage, got %d non-stage", len(ue))
	}

	stages := config.FilterWithType[*internalversion.Stage](sio)
	if len(stages) == 0 {
		return nil, fmt.Errorf("expected at least one stage, got 0")
	}

	target := rio[0]
	testTarget, ok := target.(obj)
	if !ok {
		return nil, fmt.Errorf("expected target to be an object, got %T", target)
	}

	gvk := testTarget.GetObjectKind().GroupVersionKind()

	want := internalversion.StageResourceRef{
		APIGroup: gvk.GroupVersion().String(),
		Kind:     gvk.Kind,
	}

	stages = slices.Filter(stages, func(stage *internalversion.Stage) bool {
		return stage.Spec.ResourceRef == want
	})

	lc, err := lifecycle.NewLifecycle(stages)
	if err != nil {
		return nil, err
	}

	event := &lifecycle.Event{
		Labels:      testTarget.GetLabels(),
		Annotations: testTarget.GetAnnotations(),
		Data:        testTarget,
	}

	return func() (*lifecycle.Stage, error) {
		return lc.Match(ctx, event)
	}, nil
}

var now = time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)

func testingStage(ctx context.Context, testTarget obj, event *lifecycle.Event, stage *lifecycle.Stage) (any, error) {
	meta := map[string]any{
		"stage": stage.Name(),
	}

	delay, ok, err := stage.DelayRangePossible(ctx, event, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get delay for stage %s: %w", stage.Name(), err)
	}
	if ok {
		meta["delay"] = delay
	}

	weight, ok, err := stage.Weight(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to get weight for stage %s: %w", stage.Name(), err)
	}
	if ok {
		meta["weight"] = weight
	}

	out := make([]any, 0)

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

	_, err = stage.DoSteps(0, testTarget.GetFinalizers(), testTarget, renderer,
		func(event *internalversion.StageEvent) error {
			out = append(out, formatEvent(event))
			return nil
		},
		func() error {
			out = append(out, map[string]string{
				"kind": "delete",
			})
			return nil
		},
		func(patch *lifecycle.Patch) error {
			out = append(out, formatPatch(patch))
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to do steps for stage %s: %w", stage.Name(), err)
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

func formatEvent(event *internalversion.StageEvent) any {
	out := map[string]any{
		"kind":    "event",
		"type":    event.Type,
		"reason":  event.Reason,
		"message": event.Message,
	}
	return out
}

type obj interface {
	metav1.Object
	runtime.Object
}
