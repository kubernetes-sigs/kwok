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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
)

func TestLifecycleJitterDurationFrom(t *testing.T) {
	baseMilliseconds := int64(1000)
	fixedMilliseconds := int64(3000)
	tests := []struct {
		name  string
		from  *internalversion.ExpressionFrom
		fixed *int64
		want  []time.Duration
	}{
		{
			name: "CEL duration",
			from: &internalversion.ExpressionFrom{CEL: &internalversion.ExpressionCEL{Expression: `duration("7s")`}},
			want: []time.Duration{time.Second, 7 * time.Second},
		},
		{
			name: "CEL annotation",
			from: &internalversion.ExpressionFrom{CEL: &internalversion.ExpressionCEL{Expression: `duration(self.metadata.annotations["jitter"])`}},
			want: []time.Duration{time.Second, 7 * time.Second},
		},
		{
			name:  "CEL overrides fixed jitter",
			from:  &internalversion.ExpressionFrom{CEL: &internalversion.ExpressionCEL{Expression: `duration("7s")`}},
			fixed: &fixedMilliseconds,
			want:  []time.Duration{time.Second, 7 * time.Second},
		},
		{
			name: "CEL jitter below base duration",
			from: &internalversion.ExpressionFrom{CEL: &internalversion.ExpressionCEL{Expression: `duration("500ms")`}},
			want: []time.Duration{500 * time.Millisecond},
		},
		{
			name: "JQ annotation",
			from: &internalversion.ExpressionFrom{JQ: &internalversion.ExpressionJQ{Expression: `.metadata.annotations["jitter"]`}},
			want: []time.Duration{time.Second, 7 * time.Second},
		},
		{
			name:  "fixed jitter",
			fixed: &fixedMilliseconds,
			want:  []time.Duration{time.Second, 3 * time.Second},
		},
		{
			name: "no jitter",
			want: []time.Duration{time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages, err := lifecycle.NewLifecycle([]*internalversion.Stage{{
				Spec: internalversion.StageSpec{
					Selector: &internalversion.StageSelector{},
					Delay: &internalversion.StageDelay{
						DurationMilliseconds:       &baseMilliseconds,
						JitterDurationMilliseconds: tt.fixed,
						JitterDurationFrom:         tt.from,
					},
				},
			}})
			if err != nil {
				t.Fatal(err)
			}
			event := &lifecycle.Event{Data: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"jitter": "7s"},
			}}}
			stage, err := stages.Match(context.Background(), event)
			if err != nil {
				t.Fatal(err)
			}
			if stage == nil {
				t.Fatal("stage did not match")
			}
			got, ok, err := stage.DelayRangePossible(context.Background(), event, time.Unix(0, 0))
			if err != nil {
				t.Fatal(err)
			}
			if !ok || !slices.Equal(got, tt.want) {
				t.Fatalf("DelayRangePossible() = %v, %v; want %v, true", got, ok, tt.want)
			}
		})
	}
}
