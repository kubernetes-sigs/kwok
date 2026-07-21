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

package cel

import (
	"testing"

	"github.com/google/cel-go/common/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestPodRequests(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "resizing",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					}},
				},
				{
					Name: "fallback",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					}},
				},
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "resizing",
					Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("250m"),
					}},
					AllocatedResources: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("300m"),
					},
				},
			},
		},
	}

	desired := PodRequests(pod, false)
	if got := desired.Cpu().String(); got != "1100m" {
		t.Fatalf("desired CPU = %s, want 1100m", got)
	}
	current := PodRequests(pod, true)
	if got := current.Cpu().String(); got != "400m" {
		t.Fatalf("current CPU = %s, want 400m", got)
	}

	got := runAndCheckExpressionWithData(t,
		`PodRequests(pod)["cpu"] == Quantity("1100m") && PodRequests(pod, true)["cpu"] == Quantity("400m")`,
		map[string]any{"pod": pod},
		types.BoolType,
	)
	if got != types.True {
		t.Fatalf("PodRequests CEL expression = %v, want true", got)
	}
}

func TestPodRequestsInitSidecarAndOverhead(t *testing.T) {
	restartAlways := corev1.ContainerRestartPolicyAlways
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				}},
			}},
			InitContainers: []corev1.Container{
				{
					Name:          "sidecar",
					RestartPolicy: &restartAlways,
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					}},
				},
				{
					Name: "init",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					}},
				},
			},
			Overhead: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m")},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "app",
				Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("80m"),
				}},
			}},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "sidecar",
					Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("150m"),
					}},
				},
				{
					Name: "init",
					Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("50m"),
					}},
				},
			},
		},
	}

	desired := PodRequests(pod, false)
	if got := desired.Cpu().String(); got != "1250m" {
		t.Fatalf("desired CPU = %s, want 1250m", got)
	}
	current := PodRequests(pod, true)
	if got := current.Cpu().String(); got != "1200m" {
		t.Fatalf("current CPU = %s, want 1200m", got)
	}
}

func TestPodRequestsPodLevelResourcesAndOverhead(t *testing.T) {
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				}},
			}},
			Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("2"),
			}},
			Overhead: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m")},
		},
		Status: corev1.PodStatus{
			Resources: &corev1.ResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("1"),
			}},
		},
	}

	desired := PodRequests(pod, false)
	if got := desired.Cpu().String(); got != "2050m" {
		t.Fatalf("desired CPU = %s, want 2050m", got)
	}
	current := PodRequests(pod, true)
	if got := current.Cpu().String(); got != "1050m" {
		t.Fatalf("current CPU = %s, want 1050m", got)
	}
}
