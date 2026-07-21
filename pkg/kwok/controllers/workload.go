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
	"sort"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/kwok/pkg/kwok/metrics"
	"sigs.k8s.io/kwok/pkg/utils/informer"
)

type workloadResolver struct {
	mutex sync.Mutex

	replicaSets  informer.Getter[*appsv1.ReplicaSet]
	deployments  informer.Getter[*appsv1.Deployment]
	statefulSets informer.Getter[*appsv1.StatefulSet]
	daemonSets   informer.Getter[*appsv1.DaemonSet]
	jobs         informer.Getter[*batchv1.Job]
	cronJobs     informer.Getter[*batchv1.CronJob]

	pods         map[string]*corev1.Pod
	resolved     map[string]*metrics.Workload
	version      int64
	cacheVersion int64
}

func newWorkloadResolver(
	replicaSets informer.Getter[*appsv1.ReplicaSet],
	deployments informer.Getter[*appsv1.Deployment],
	statefulSets informer.Getter[*appsv1.StatefulSet],
	daemonSets informer.Getter[*appsv1.DaemonSet],
	jobs informer.Getter[*batchv1.Job],
	cronJobs informer.Getter[*batchv1.CronJob],
) *workloadResolver {
	return &workloadResolver{
		replicaSets:  replicaSets,
		deployments:  deployments,
		statefulSets: statefulSets,
		daemonSets:   daemonSets,
		jobs:         jobs,
		cronJobs:     cronJobs,
		pods:         map[string]*corev1.Pod{},
	}
}

func workloadPodKey(pod *corev1.Pod) string {
	if pod.UID != "" {
		return string(pod.UID)
	}
	return pod.Namespace + "/" + pod.Name
}

func workloadPodActive(pod *corev1.Pod) bool {
	return pod.DeletionTimestamp == nil &&
		pod.Spec.NodeName != "" &&
		pod.Status.Phase != corev1.PodSucceeded &&
		pod.Status.Phase != corev1.PodFailed &&
		len(pod.Spec.Containers) != 0
}

func (r *workloadResolver) updatePod(pod *corev1.Pod) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	key := workloadPodKey(pod)
	if workloadPodActive(pod) {
		r.pods[key] = pod.DeepCopy()
	} else {
		delete(r.pods, key)
	}
	r.version++
}

func (r *workloadResolver) deletePod(pod *corev1.Pod) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pods, workloadPodKey(pod))
	r.version++
}

func (r *workloadResolver) invalidate() {
	r.mutex.Lock()
	r.version++
	r.mutex.Unlock()
}

type workloadIdentity struct {
	apiVersion string
	kind       string
	namespace  string
	name       string
	uid        string
	created    metav1.Time
}

func ownerIdentity(namespace string, owner *metav1.OwnerReference, created metav1.Time) workloadIdentity {
	return workloadIdentity{
		apiVersion: owner.APIVersion,
		kind:       owner.Kind,
		namespace:  namespace,
		name:       owner.Name,
		uid:        string(owner.UID),
		created:    created,
	}
}

func podIdentity(pod *corev1.Pod) workloadIdentity {
	return workloadIdentity{
		apiVersion: corev1.SchemeGroupVersion.String(),
		kind:       "Pod",
		namespace:  pod.Namespace,
		name:       pod.Name,
		uid:        string(pod.UID),
		created:    pod.CreationTimestamp,
	}
}

func (r *workloadResolver) identityForPod(pod *corev1.Pod) workloadIdentity {
	owner := metav1.GetControllerOf(pod)
	if owner == nil {
		return podIdentity(pod)
	}
	identity := ownerIdentity(pod.Namespace, owner, pod.CreationTimestamp)
	switch {
	case owner.APIVersion == appsv1.SchemeGroupVersion.String() && owner.Kind == "ReplicaSet":
		if replicaSet, ok := r.replicaSets.GetWithNamespace(owner.Name, pod.Namespace); ok {
			identity.created = replicaSet.CreationTimestamp
			if parent := metav1.GetControllerOf(replicaSet); parent != nil &&
				parent.APIVersion == appsv1.SchemeGroupVersion.String() && parent.Kind == "Deployment" {
				created := replicaSet.CreationTimestamp
				if deployment, ok := r.deployments.GetWithNamespace(parent.Name, pod.Namespace); ok {
					created = deployment.CreationTimestamp
				}
				return ownerIdentity(pod.Namespace, parent, created)
			}
		}
	case owner.APIVersion == appsv1.SchemeGroupVersion.String() && owner.Kind == "StatefulSet":
		if statefulSet, ok := r.statefulSets.GetWithNamespace(owner.Name, pod.Namespace); ok {
			identity.created = statefulSet.CreationTimestamp
		}
	case owner.APIVersion == appsv1.SchemeGroupVersion.String() && owner.Kind == "DaemonSet":
		if daemonSet, ok := r.daemonSets.GetWithNamespace(owner.Name, pod.Namespace); ok {
			identity.created = daemonSet.CreationTimestamp
		}
	case owner.APIVersion == batchv1.SchemeGroupVersion.String() && owner.Kind == "Job":
		if job, ok := r.jobs.GetWithNamespace(owner.Name, pod.Namespace); ok {
			identity.created = job.CreationTimestamp
			if parent := metav1.GetControllerOf(job); parent != nil &&
				parent.APIVersion == batchv1.SchemeGroupVersion.String() && parent.Kind == "CronJob" {
				created := job.CreationTimestamp
				if cronJob, ok := r.cronJobs.GetWithNamespace(parent.Name, pod.Namespace); ok {
					created = cronJob.CreationTimestamp
				}
				return ownerIdentity(pod.Namespace, parent, created)
			}
		}
	}
	return identity
}

func workloadIdentityKey(identity workloadIdentity) string {
	if identity.uid != "" {
		return identity.apiVersion + "/" + identity.kind + "/" + identity.uid
	}
	return identity.apiVersion + "/" + identity.kind + "/" + identity.namespace + "/" + identity.name
}

func workloadPodLess(a, b *corev1.Pod) bool {
	if a.UID != b.UID {
		return string(a.UID) < string(b.UID)
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Name < b.Name
}

func (r *workloadResolver) rebuild() {
	groups := map[string][]*corev1.Pod{}
	identities := map[string]workloadIdentity{}
	for _, pod := range r.pods {
		identity := r.identityForPod(pod)
		key := workloadIdentityKey(identity)
		identities[key] = identity
		groups[key] = append(groups[key], pod)
	}

	resolved := make(map[string]*metrics.Workload, len(r.pods))
	for key, pods := range groups {
		sort.Slice(pods, func(i, j int) bool { return workloadPodLess(pods[i], pods[j]) })
		identity := identities[key]
		created := identity.created
		if created.IsZero() {
			created = pods[0].CreationTimestamp
		}
		for index, pod := range pods {
			resolved[workloadPodKey(pod)] = &metrics.Workload{
				APIVersion:        identity.apiVersion,
				Kind:              identity.kind,
				Namespace:         identity.namespace,
				Name:              identity.name,
				UID:               identity.uid,
				Replicas:          int64(len(pods)),
				PodIndex:          int64(index),
				CreationTimestamp: created,
				Version:           r.version,
			}
		}
	}
	r.resolved = resolved
	r.cacheVersion = r.version
}

func (r *workloadResolver) ResolveWorkload(pod *corev1.Pod) *metrics.Workload {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.cacheVersion != r.version || r.resolved == nil {
		r.rebuild()
	}
	workload, ok := r.resolved[workloadPodKey(pod)]
	if !ok {
		return metrics.WorkloadForPod(pod)
	}
	copy := *workload
	return &copy
}
