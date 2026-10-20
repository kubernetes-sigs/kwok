/*
Copyright 2026 The Kubernetes Authors.

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

package lifecycle_test

import (
	"context"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
)

func TestLifecycleWeightFromJQ(t *testing.T) {
	for _, tt := range []struct {
		name       string
		expression string
		wantWeight int64
		wantNames  []string
	}{
		{"constant zero", "0", 0, []string{"fallback"}},
		{"arithmetic zero", "1 - 1", 0, []string{"fallback"}},
		{"parsed zero", ".metadata.annotations.weight | tonumber", 0, []string{"fallback"}},
		{"positive integer", "7", 7, []string{"dynamic", "fallback"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lc, err := lifecycle.NewLifecycle([]*internalversion.Stage{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "dynamic"},
					Spec: internalversion.StageSpec{
						Selector: &internalversion.StageSelector{},
						Weight:   99,
						WeightFrom: &internalversion.ExpressionFrom{
							JQ: &internalversion.ExpressionJQ{Expression: tt.expression},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "fallback"},
					Spec: internalversion.StageSpec{
						Selector: &internalversion.StageSelector{},
						Weight:   1,
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			event := &lifecycle.Event{Data: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"weight": "0"}},
			}}
			stages, err := lc.ListAllPossible(context.Background(), event)
			if err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(stages))
			for _, stage := range stages {
				names = append(names, stage.Name())
			}
			if !slices.Equal(names, tt.wantNames) {
				t.Errorf("possible stages = %v, want %v", names, tt.wantNames)
			}
			weight, ok, err := lc[0].Weight(context.Background(), event)
			if err != nil || !ok || weight != tt.wantWeight {
				t.Errorf("dynamic weight = (%d, %t, %v), want (%d, true, nil)", weight, ok, err, tt.wantWeight)
			}
		})
	}
}
