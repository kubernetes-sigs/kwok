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

package lifecycle

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	utilscel "sigs.k8s.io/kwok/pkg/utils/cel"
)

func TestLifecycleCELVariables(t *testing.T) {
	stage := NewStage(&internalversion.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "fits"},
		Spec: internalversion.StageSpec{
			Selector: &internalversion.StageSelector{
				MatchExpressions: []internalversion.MatchExpression{{CEL: &internalversion.ExpressionCEL{
					Expression: `node.metadata.name == "node-a" && nodeAllocatedRequests["cpu"] == Quantity("500m") && PodRequests(self)["cpu"] == Quantity("1")`,
				}}},
			},
		},
	})
	if !stage.UsesPodResourceContext() {
		t.Fatal("pod resource context stage was not detected")
	}
	nonContextualStage := NewStage(&internalversion.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: "non-contextual"},
		Spec: internalversion.StageSpec{Selector: &internalversion.StageSelector{
			MatchExpressions: []internalversion.MatchExpression{{CEL: &internalversion.ExpressionCEL{
				Expression: `self.spec.nodeName == "node-a"`,
			}}},
		}},
	})
	if nonContextualStage.UsesPodResourceContext() {
		t.Fatal("non-contextual CEL stage was detected as pod resource context")
	}

	pod := &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("1"),
		}},
	}}}}
	event := &Event{
		Data: pod,
		CELVariables: map[string]any{
			"node": &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-a"}},
			"nodeAllocatedRequests": utilscel.NewResourceList(corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			}),
		},
	}
	matched, err := Lifecycle{stage}.Match(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if matched != stage {
		t.Fatalf("matched stage = %v, want %s", matched, stage.Name())
	}
}

func TestLifecycleNilCELVariable(t *testing.T) {
	event := &Event{Data: &corev1.Pod{}, CELVariables: map[string]any{"optional": nil}}
	if _, ok := event.toCELStandardTypeOnly()["optional"]; ok {
		t.Fatal("nil optional CEL variable should not be registered")
	}
}

func TestPodResizeCapacityClassification(t *testing.T) {
	const (
		infeasible = `PodRequests(self)["cpu"] > node.status.allocatable["cpu"]`
		deferred   = `PodRequests(self)["cpu"] <= node.status.allocatable["cpu"] && nodeAllocatedRequests["cpu"] - PodRequests(self, true)["cpu"] + PodRequests(self)["cpu"] > node.status.allocatable["cpu"]`
		success    = `PodRequests(self)["cpu"] <= node.status.allocatable["cpu"] && nodeAllocatedRequests["cpu"] - PodRequests(self, true)["cpu"] + PodRequests(self)["cpu"] <= node.status.allocatable["cpu"]`
	)
	stages := []*internalversion.Stage{
		capacityStage("infeasible", infeasible),
		capacityStage("deferred", deferred),
		capacityStage("success", success),
	}
	lifecycle, err := NewLifecycle(stages)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name        string
		allocatable string
		allocated   string
		want        string
	}{
		{name: "success", allocatable: "2", allocated: "250m", want: "success"},
		{name: "deferred", allocatable: "1", allocated: "500m", want: "deferred"},
		{name: "infeasible", allocatable: "800m", allocated: "250m", want: "infeasible"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pod := resizePod("960m", "250m")
			node := &corev1.Node{Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(tt.allocatable),
			}}}
			event := &Event{
				Data: pod,
				CELVariables: map[string]any{
					"node": node,
					"nodeAllocatedRequests": utilscel.NewResourceList(corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse(tt.allocated),
					}),
				},
			}
			matched, err := lifecycle.Match(context.Background(), event)
			if err != nil {
				t.Fatal(err)
			}
			if matched == nil || matched.Name() != tt.want {
				t.Fatalf("matched stage = %v, want %s", matched, tt.want)
			}
		})
	}
}

func capacityStage(name, expression string) *internalversion.Stage {
	return &internalversion.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: internalversion.StageSpec{Selector: &internalversion.StageSelector{
			MatchExpressions: []internalversion.MatchExpression{{CEL: &internalversion.ExpressionCEL{Expression: expression}}},
		}},
	}
}

func resizePod(desiredCPU, currentCPU string) *corev1.Pod {
	return &corev1.Pod{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "container",
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(desiredCPU),
			}},
		}}},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name: "container",
			Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(currentCPU),
			}},
		}}},
	}
}
