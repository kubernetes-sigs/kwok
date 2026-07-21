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

package controllers

import (
	"context"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	fakeclock "k8s.io/utils/clock/testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
	"sigs.k8s.io/kwok/pkg/utils/queue"
)

func TestPodAllocationLedger(t *testing.T) {
	controller := &PodController{}
	pod1 := allocatedPod("pod-1", "node-a", "250m", "250m")
	pod2 := allocatedPod("pod-2", "node-a", "250m", "250m")
	controller.putPodInfo(pod1)
	controller.putPodInfo(pod2)

	requests := controller.nodeAllocatedRequests("node-a")
	if got := requests.Cpu().String(); got != "500m" {
		t.Fatalf("allocated CPU = %s, want 500m", got)
	}
	pod2.Status.Phase = corev1.PodSucceeded
	if !controller.putPodInfo(pod2) {
		t.Fatal("terminal pod should release its allocation")
	}
	requests = controller.nodeAllocatedRequests("node-a")
	if got := requests.Cpu().String(); got != "250m" {
		t.Fatalf("allocated CPU after completion = %s, want 250m", got)
	}
}

func TestDeferredRetryScheduling(t *testing.T) {
	clock := fakeclock.NewFakeClock(time.Now())
	controller := &PodController{
		clock:         clock,
		deferredQueue: queue.NewDelayingQueue[deferredRetryJob](clock),
	}
	pod := allocatedPod("pod", "node-a", "750m", "250m")
	pod.Generation = 2
	pod.Status.Conditions = []corev1.PodCondition{{
		Type:               corev1.PodResizePending,
		Status:             corev1.ConditionTrue,
		Reason:             corev1.PodReasonDeferred,
		ObservedGeneration: 2,
	}}
	controller.putPodInfo(pod)
	controller.ensureDeferredRetry(pod, 2)

	waitForFakeClockWaiters(t, clock, 2)
	clock.Step(deferredInitialRetryDelay - time.Second)
	if _, ok := controller.deferredQueue.Get(); ok {
		t.Fatal("deferred retry became ready before 30 seconds")
	}
	clock.Step(time.Second)
	waitForQueueItem(t, controller.deferredQueue)

	controller.scheduleDeferredRetry(log.KObj(pod), 2, deferredPeriodicRetryDelay)
	waitForFakeClockWaiters(t, clock, 2)
	clock.Step(deferredPeriodicRetryDelay - time.Second)
	if _, ok := controller.deferredQueue.Get(); ok {
		t.Fatal("periodic deferred retry became ready before three minutes")
	}
	clock.Step(time.Second)
	waitForQueueItem(t, controller.deferredQueue)

	controller.scheduleDeferredRetry(log.KObj(pod), 2, deferredPeriodicRetryDelay)
	controller.NodeAllocatableChanged("node-a")
	job := waitForQueueItem(t, controller.deferredQueue)
	state, ok := controller.deferredRetries.Load(log.KObj(pod))
	if !ok || job.Token != state.Token {
		t.Fatal("node capacity change did not schedule the current deferred retry immediately")
	}
}

func TestConcurrentCapacityStagesRevalidatePerNode(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
		Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("1"),
		}},
	}
	pod1 := allocatedPod("pod-1", node.Name, "750m", "250m")
	pod2 := allocatedPod("pod-2", node.Name, "750m", "250m")
	client := fake.NewClientset(pod1, pod2)
	lc, err := lifecycle.NewLifecycle(resizeCapacityStages())
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewPodController(PodControllerConfig{
		TypedClient:          client,
		NodeCacheGetter:      staticNodeGetter{node: node},
		NodeGetFunc:          func(string) (*NodeInfo, bool) { return &NodeInfo{}, true },
		Lifecycle:            resources.NewStaticGetter(lc),
		PlayStageParallelism: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.putPodInfo(pod1)
	controller.putPodInfo(pod2)
	initialStage, err := lc.Match(context.Background(), controller.lifecycleEvent(pod1))
	if err != nil {
		t.Fatal(err)
	}
	if initialStage == nil || initialStage.Name() != "success" {
		t.Fatalf("initial stage = %v, want success", initialStage)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, pod := range []*corev1.Pod{pod1, pod2} {
		wg.Go(func() {
			_, err := controller.playStage(context.Background(), pod, initialStage, 0)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	var successCount, deferredCount int
	for _, name := range []string{pod1.Name, pod2.Name} {
		pod, err := client.CoreV1().Pods("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if pod.Status.ContainerStatuses[0].Resources.Requests.Cpu().String() == "750m" {
			successCount++
		}
		if _, deferred := podDeferredGeneration(pod); deferred {
			deferredCount++
		}
	}
	if successCount != 1 || deferredCount != 1 {
		t.Fatalf("success/deferred counts = %d/%d, want 1/1", successCount, deferredCount)
	}
}

func TestNodeAllocatableChangeDetection(t *testing.T) {
	controller := &NodeController{}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
		Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("1"),
		}},
	}
	if controller.putNodeInfo(node) {
		t.Fatal("initial node observation should not report a capacity change")
	}
	node.Status.Allocatable[corev1.ResourceCPU] = resource.MustParse("2")
	if !controller.putNodeInfo(node) {
		t.Fatal("allocatable update was not detected")
	}
}

func TestNodeAllocatableChangeRetriesInfeasiblePod(t *testing.T) {
	clock := fakeclock.NewFakeClock(time.Now())
	controller, err := NewPodController(PodControllerConfig{
		Clock:                clock,
		PlayStageParallelism: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	pod := allocatedPod("pod", "node-a", "1", "250m")
	pod.Status.Conditions = []corev1.PodCondition{{
		Type:               corev1.PodResizePending,
		Status:             corev1.ConditionTrue,
		Reason:             corev1.PodReasonInfeasible,
		ObservedGeneration: pod.Generation,
	}}
	controller.putPodInfo(pod)

	controller.NodeAllocatableChanged(pod.Spec.NodeName)
	ref, ok := controller.contextRetryQueue.Get()
	if !ok || ref != log.KObj(pod) {
		t.Fatalf("infeasible retry = %v, %v; want %v, true", ref, ok, log.KObj(pod))
	}
}

func TestMissingNodeContextSchedulesRetry(t *testing.T) {
	clock := fakeclock.NewFakeClock(time.Now())
	pod := allocatedPod("pod", "missing-node", "250m", "250m")
	controller, err := NewPodController(PodControllerConfig{
		Clock:                clock,
		TypedClient:          fake.NewClientset(pod),
		NodeCacheGetter:      staticNodeGetter{node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "other-node"}}},
		PlayStageParallelism: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.preprocess(context.Background(), pod); err == nil {
		t.Fatal("missing node context should return an error")
	}
	clock.Step(time.Second)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ref, ok := controller.contextRetryQueue.Get(); ok {
			if ref != log.KObj(pod) {
				t.Fatalf("retry ref = %v, want %v", ref, log.KObj(pod))
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("missing node context did not schedule a retry")
}

func waitForQueueItem(t *testing.T, queue interface {
	Get() (deferredRetryJob, bool)
}) deferredRetryJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if item, ok := queue.Get(); ok {
			return item
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for deferred retry")
	return deferredRetryJob{}
}

func waitForFakeClockWaiters(t *testing.T, clock *fakeclock.FakeClock, count int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if clock.Waiters() >= count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d deferred retry timers", count)
}

type staticNodeGetter struct {
	node *corev1.Node
}

func (g staticNodeGetter) Get(name string) (*corev1.Node, bool) {
	if g.node.Name != name {
		return nil, false
	}
	return g.node, true
}

func (g staticNodeGetter) GetWithNamespace(name, _ string) (*corev1.Node, bool) {
	return g.Get(name)
}

func (g staticNodeGetter) List() []*corev1.Node {
	return []*corev1.Node{g.node}
}

func allocatedPod(name, node, desiredCPU, currentCPU string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Generation: 1},
		Spec: corev1.PodSpec{
			NodeName: node,
			Containers: []corev1.Container{{
				Name: "container",
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse(desiredCPU),
				}},
			}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "container",
				Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse(currentCPU),
				}},
			}},
		},
	}
}

func resizeCapacityStages() []*internalversion.Stage {
	return []*internalversion.Stage{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "deferred"},
			Spec: internalversion.StageSpec{
				Selector: &internalversion.StageSelector{MatchExpressions: []internalversion.MatchExpression{{CEL: &internalversion.ExpressionCEL{
					Expression: `PodRequests(self)["cpu"] <= node.status.allocatable["cpu"] && nodeAllocatedRequests["cpu"] - PodRequests(self, true)["cpu"] + PodRequests(self)["cpu"] > node.status.allocatable["cpu"]`,
				}}}},
				Steps: []internalversion.StageStep{{Patch: &internalversion.StagePatch{
					Root:        "status",
					Subresource: "status",
					Template: `conditions:
- type: PodResizePending
  status: "True"
  reason: Deferred
  observedGeneration: {{ .metadata.generation }}`,
				}}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "success"},
			Spec: internalversion.StageSpec{
				Selector: &internalversion.StageSelector{MatchExpressions: []internalversion.MatchExpression{{CEL: &internalversion.ExpressionCEL{
					Expression: `PodRequests(self)["cpu"] <= node.status.allocatable["cpu"] && nodeAllocatedRequests["cpu"] - PodRequests(self, true)["cpu"] + PodRequests(self)["cpu"] <= node.status.allocatable["cpu"]`,
				}}}},
				Steps: []internalversion.StageStep{{Patch: &internalversion.StagePatch{
					Root:        "status",
					Subresource: "status",
					Template: `containerStatuses:
{{ range .spec.containers }}
- name: {{ .name }}
  resources:
    requests:
      cpu: {{ index .resources.requests "cpu" | Quote }}
{{ end }}`,
				}}},
			},
		},
	}
}
