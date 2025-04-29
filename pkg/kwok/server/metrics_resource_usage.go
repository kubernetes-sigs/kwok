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

package server

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwok/metrics"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

type cumulative struct {
	time  time.Time
	value float64
}

func (s *Server) containerResourceCumulativeUsage(resourceName, podNamespace, podName, containerName string) float64 {
	key := fmt.Sprintf("%s/%s/%s/%s", resourceName, podNamespace, podName, containerName)
	v := s.containerResourceUsage(resourceName, podNamespace, podName, containerName)

	now := time.Now()
	s.cumulativesMut.Lock()
	defer s.cumulativesMut.Unlock()
	c, ok := s.cumulatives[key]
	if ok {
		c.value += now.Sub(c.time).Seconds() * v
	}

	c.time = now
	s.cumulatives[key] = c

	return c.value
}

func (s *Server) podResourceCumulativeUsage(resourceName, podNamespace, podName string) float64 {
	pod, ok := s.podCacheGetter.GetWithNamespace(podName, podNamespace)
	if !ok {
		return 0
	}

	sum := 0.0
	for _, c := range pod.Spec.Containers {
		sum += s.containerResourceCumulativeUsage(resourceName, podNamespace, podName, c.Name)
	}
	return sum
}

func (s *Server) nodeResourceCumulativeUsage(resourceName, nodeName string) float64 {
	node, ok := s.nodeCacheGetter.Get(nodeName)
	if !ok {
		return 0
	}

	pods, ok := s.dataSource.ListPods(nodeName)
	if !ok {
		return 0
	}

	data := metrics.Data{
		Node: node,
	}

	sum := 0.0
	for _, pi := range pods {
		pod, ok := s.podCacheGetter.GetWithNamespace(pi.Name, pi.Namespace)
		if !ok {
			continue
		}
		data.Pod = pod
		for _, c := range pod.Spec.Containers {
			data.Container = &c
			sum += s.evaluateContainerResourceUsage(resourceName, data)
		}
	}

	now := time.Now()
	key := nodeName
	s.cumulativesMut.Lock()
	defer s.cumulativesMut.Unlock()
	c, ok := s.cumulatives[key]
	if ok {
		c.value += now.Sub(c.time).Seconds() * sum
	}

	c.time = now
	s.cumulatives[key] = c

	return c.value
}

func (s *Server) containerResourceUsage(resourceName, podNamespace, podName, containerName string) float64 {
	pod, ok := s.podCacheGetter.GetWithNamespace(podName, podNamespace)
	if !ok {
		return 0
	}

	node, ok := s.nodeCacheGetter.Get(pod.Spec.NodeName)
	if !ok {
		return 0
	}

	c, ok := slices.Find(pod.Spec.Containers, func(c corev1.Container) bool {
		return c.Name == containerName
	})
	if !ok {
		return 0
	}
	data := metrics.Data{
		Node:      node,
		Pod:       pod,
		Container: &c,
	}
	return s.evaluateContainerResourceUsage(resourceName, data)
}

func (s *Server) evaluateContainerResourceUsage(resourceName string, data metrics.Data) float64 {
	u, err := s.getResourceUsage(data.Pod.Name, data.Pod.Namespace, data.Container.Name)
	if err != nil {
		logger := log.FromContext(s.ctx)
		logger.Error("failed to get resource usage", err, "pod", log.KRef(data.Pod.Namespace, data.Pod.Name), "container", data.Container.Name)
		return 0
	}
	if u.Usage == nil {
		return 0
	}
	r := u.Usage[resourceName]
	if r.Value != nil {
		return r.Value.AsApproximateFloat64()
	}

	if r.Expression != nil {
		eval, err := s.env.Compile(*r.Expression)
		if err != nil {
			logger := log.FromContext(s.ctx)
			logger.Error("failed to compile expression", err, "expression", r)
			return 0
		}

		out, err := eval.EvaluateFloat64(s.ctx, data)
		if err != nil {
			logger := log.FromContext(s.ctx)
			logger.Error("failed to evaluate expression", err, "expression", r)
			return 0
		}
		return out
	}
	return 0
}

func (s *Server) podResourceUsage(resourceName, podNamespace, podName string) float64 {
	pod, ok := s.podCacheGetter.GetWithNamespace(podName, podNamespace)
	if !ok {
		return 0
	}

	node, ok := s.nodeCacheGetter.Get(pod.Spec.NodeName)
	if !ok {
		return 0
	}

	data := metrics.Data{
		Node: node,
		Pod:  pod,
	}

	sum := 0.0
	for _, c := range pod.Spec.Containers {
		data.Container = &c
		sum += s.evaluateContainerResourceUsage(resourceName, data)
	}
	return sum
}

func (s *Server) nodeResourceUsage(resourceName, nodeName string) float64 {
	node, ok := s.nodeCacheGetter.Get(nodeName)
	if !ok {
		return 0
	}

	pods, ok := s.dataSource.ListPods(nodeName)
	if !ok {
		return 0
	}

	data := metrics.Data{
		Node: node,
	}

	sum := 0.0
	for _, pi := range pods {
		pod, ok := s.podCacheGetter.GetWithNamespace(pi.Name, pi.Namespace)
		if !ok {
			continue
		}
		data.Pod = pod
		for _, c := range pod.Spec.Containers {
			data.Container = &c
			sum += s.evaluateContainerResourceUsage(resourceName, data)
		}
	}
	return sum
}

func (s *Server) getResourceUsage(podName, podNamespace, containerName string) (*internalversion.ResourceUsageContainer, error) {
	u, has := slices.Find(s.resourceUsages.Get(), func(a *internalversion.ResourceUsage) bool {
		return a.Name == podName && a.Namespace == podNamespace
	})
	if has {
		u, found := findUsageInUsages(containerName, u.Spec.Usages)
		if found {
			return u, nil
		}
		return nil, fmt.Errorf("no resource usage found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
	}

	for _, cl := range s.clusterResourceUsages.Get() {
		if !cl.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		u, found := findUsageInUsages(containerName, cl.Spec.Usages)
		if found {
			return u, nil
		}
	}

	return nil, fmt.Errorf("no resource usage found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
}

func findUsageInUsages(containerName string, usages []internalversion.ResourceUsageContainer) (*internalversion.ResourceUsageContainer, bool) {
	var defaultUsage *internalversion.ResourceUsageContainer
	for i, u := range usages {
		if len(u.Containers) == 0 && defaultUsage == nil {
			defaultUsage = &usages[i]
			continue
		}
		if slices.Contains(u.Containers, containerName) {
			return &u, true
		}
	}
	return defaultUsage, defaultUsage != nil
}
