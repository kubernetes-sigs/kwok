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

package cel

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeEvaluation(t *testing.T) {
	now := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
		},
	}
	exp := "( now().unixSecond() - node.metadata.creationTimestamp.unixSecond() ) * node.startedContainersTotal() / 10"

	env, err := NewEnvironment(NodeEvaluatorConfig{
		Now: func() time.Time {
			return now
		},
		StartedContainersTotal: func(nodeName string) int64 {
			return 2
		},
	})
	if err != nil {
		t.Fatalf("failed to instantiate node Evaluator: %v", err)
	}

	eval, err := env.Compile(exp)
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	actual, err := eval.EvaluateFloat64(n)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if actual != 17280 {
		t.Errorf("expected %v, got %v", 17280, actual)
	}
}
