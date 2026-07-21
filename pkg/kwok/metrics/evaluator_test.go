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

package metrics

import (
	"context"
	"math"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeEvaluation(t *testing.T) {
	now := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: now.Add(-24 * time.Hour)},
		},
	}
	exp := "( Now().UnixSecond() - node.metadata.creationTimestamp.UnixSecond() ) * node.StartedContainersTotal() / 10.0"

	env, err := NewEnvironment(EnvironmentConfig{
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

	actual, err := eval.EvaluateFloat64(context.Background(), Data{
		Node: n,
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if actual != 17280 {
		t.Errorf("expected %v, got %v", 17280, actual)
	}
}

func TestResourceEvaluation(t *testing.T) {
	n := &corev1.Node{
		Status: corev1.NodeStatus{
			Allocatable: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: resource.MustParse("90m"),
			},
		},
	}
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"cpu_usage": "10m",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceCPU: resource.MustParse("10m"),
						},
					},
				},
			},
		},
	}

	exp := `( Quantity("10m") + Quantity(pod.metadata.annotations["cpu_usage"]) + node.status.allocatable["cpu"] + pod.spec.containers[0].resources.requests["cpu"] ) * 1.5 * 10`

	env, err := NewEnvironment(EnvironmentConfig{})
	if err != nil {
		t.Fatalf("failed to instantiate node Evaluator: %v", err)
	}

	eval, err := env.Compile(exp)
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	actual, err := eval.EvaluateFloat64(context.Background(), Data{
		Node: n,
		Pod:  p,
	})
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	const epsilon = 1e-9
	expected := 1.8
	if math.Abs(actual-expected) > epsilon {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func TestWorkloadEvaluation(t *testing.T) {
	created := metav1.NewTime(time.Now().Add(-time.Minute))
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default", UID: "pod-uid", CreationTimestamp: created},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "container"}}},
	}
	workload := &Workload{
		APIVersion:        "apps/v1",
		Kind:              "Deployment",
		Namespace:         "default",
		Name:              "example",
		UID:               "deployment-uid",
		Replicas:          4,
		PodIndex:          2,
		CreationTimestamp: created,
		Version:           1,
	}
	env, err := NewEnvironment(EnvironmentConfig{WorkloadForPod: func(*corev1.Pod) *Workload { return workload }})
	if err != nil {
		t.Fatal(err)
	}

	eval, err := env.Compile(`double(workload.replicas * 100 + workload.podIndex) + (workload.SinceSecond() > 0.0 ? 1.0 : 0.0)`)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := eval.EvaluateFloat64(context.Background(), Data{Pod: pod})
	if err != nil {
		t.Fatal(err)
	}
	if actual != 403 {
		t.Fatalf("expected workload fields and SinceSecond to evaluate to 403, got %v", actual)
	}

	identity, err := env.Compile(`workload.apiVersion + "/" + workload.kind + "/" + workload.namespace + "/" + workload.name + "/" + workload.uid`)
	if err != nil {
		t.Fatal(err)
	}
	actualIdentity, err := identity.EvaluateString(context.Background(), Data{Pod: pod})
	if err != nil {
		t.Fatal(err)
	}
	if actualIdentity != "apps/v1/Deployment/default/example/deployment-uid" {
		t.Fatalf("unexpected workload identity %q", actualIdentity)
	}

	normalized, err := env.Compile(`800.0 / double(workload.replicas) / double(size(pod.spec.containers)) * (1.0 + double(workload.podIndex) * 0.0)`)
	if err != nil {
		t.Fatal(err)
	}
	actualNormalized, err := normalized.EvaluateFloat64(context.Background(), Data{Pod: pod})
	if err != nil || actualNormalized != 200 {
		t.Fatalf("expected normalized workload usage 200, got %v, err %v", actualNormalized, err)
	}
}

func TestWorkloadFallbackAndResultCache(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "default", UID: "pod-uid"}}
	workload := &Workload{Replicas: 2, Version: 1}
	env, err := NewEnvironment(EnvironmentConfig{
		EnableResultCache: true,
		WorkloadForPod:    func(*corev1.Pod) *Workload { return workload },
	})
	if err != nil {
		t.Fatal(err)
	}
	eval, err := env.Compile(`double(workload.replicas)`)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := eval.EvaluateFloat64(context.Background(), Data{Pod: pod})
	if err != nil || actual != 2 {
		t.Fatalf("expected initial replicas 2, got %v, err %v", actual, err)
	}
	workload = &Workload{Replicas: 3, Version: 2}
	actual, err = eval.EvaluateFloat64(context.Background(), Data{Pod: pod})
	if err != nil || actual != 3 {
		t.Fatalf("expected workload version to invalidate cached result, got %v, err %v", actual, err)
	}

	fallbackEnv, err := NewEnvironment(EnvironmentConfig{})
	if err != nil {
		t.Fatal(err)
	}
	fallbackEval, err := fallbackEnv.Compile(`workload.kind + "/" + workload.name`)
	if err != nil {
		t.Fatal(err)
	}
	actualFallback, err := fallbackEval.EvaluateString(context.Background(), Data{Pod: pod})
	if err != nil || actualFallback != "Pod/pod" {
		t.Fatalf("expected Pod compatibility workload, got %q, err %v", actualFallback, err)
	}
}
