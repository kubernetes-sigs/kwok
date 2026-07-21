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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	resourcehelper "k8s.io/component-helpers/resource"
)

func podRequests(pod *corev1.Pod) ResourceList {
	return NewResourceList(PodRequests(pod, false))
}

func podRequestsWithStatus(pod *corev1.Pod, useStatusResources bool) ResourceList {
	return NewResourceList(PodRequests(pod, useStatusResources))
}

// PodRequests returns the desired or current resource requests for a pod.
// Current requests use status resources when available and fall back to spec.
func PodRequests(pod *corev1.Pod, useStatusResources bool) corev1.ResourceList {
	if !useStatusResources {
		return resourcehelper.PodRequests(pod, resourcehelper.PodResourcesOptions{})
	}

	// PodRequests uses the maximum of desired and status resources while a resize
	// is pending. For node allocation accounting, status resources represent the
	// current allocation and spec is only a fallback when status is unavailable.
	pod = pod.DeepCopy()
	useCurrentContainerRequests(pod.Spec.Containers, pod.Status.ContainerStatuses, false)
	useCurrentContainerRequests(pod.Spec.InitContainers, pod.Status.InitContainerStatuses, true)
	if pod.Spec.Resources != nil && (pod.Status.Resources != nil || len(pod.Status.AllocatedResources) != 0) {
		if pod.Status.Resources == nil {
			pod.Status.Resources = &corev1.ResourceRequirements{}
		}
		if pod.Spec.Resources.Requests == nil {
			pod.Spec.Resources.Requests = corev1.ResourceList{}
		}
		zeroCurrentResourceNames(pod.Spec.Resources.Requests, pod.Status.Resources.Requests, pod.Status.AllocatedResources)
	}

	return resourcehelper.PodRequests(pod, resourcehelper.PodResourcesOptions{
		UseStatusResources: true,
		InPlacePodLevelResourcesVerticalScalingEnabled: true,
	})
}

func useCurrentContainerRequests(containers []corev1.Container, statuses []corev1.ContainerStatus, restartableOnly bool) {
	statusByName := make(map[string]corev1.ContainerStatus, len(statuses))
	for _, status := range statuses {
		statusByName[status.Name] = status
	}
	for i := range containers {
		if restartableOnly && (containers[i].RestartPolicy == nil || *containers[i].RestartPolicy != corev1.ContainerRestartPolicyAlways) {
			continue
		}
		status, ok := statusByName[containers[i].Name]
		if ok && (status.Resources != nil || len(status.AllocatedResources) != 0) {
			if status.Resources == nil {
				for j := range statuses {
					if statuses[j].Name == status.Name {
						statuses[j].Resources = &corev1.ResourceRequirements{}
						status = statuses[j]
						break
					}
				}
			}
			removeCurrentResourceNames(containers[i].Resources.Requests, status.Resources.Requests, status.AllocatedResources)
		}
	}
}

func removeCurrentResourceNames(spec, actuated, allocated corev1.ResourceList) {
	for name := range actuated {
		delete(spec, name)
	}
	for name := range allocated {
		delete(spec, name)
	}
}

func zeroCurrentResourceNames(spec, actuated, allocated corev1.ResourceList) {
	for name := range actuated {
		spec[name] = resource.Quantity{}
	}
	for name := range allocated {
		spec[name] = resource.Quantity{}
	}
}
