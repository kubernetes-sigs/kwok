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

package snapshot

import (
	"fmt"
)

// Resources is the resources of cluster want to save or restore
// The resource string format adheres to a GVR template.
// - a single string like "node" will be parsed as {resource: node}
// - a two-section string like "daemonset.apps" will be parsed as {resource: dasemonset, group: apps}
// - a three-section (or 3+) string like "foo.v1alpha1.example.com" will be parsed as {resource: foo, version: v1alpha1, group: example.com}
// list all resources can use: kubectl api-resources -o name
var Resources = []string{
	"namespace",
	"node",
	"serviceaccount",
	"configmap",
	"secret",
	"limitrange",
	"runtimeclass.node.k8s.io",
	"priorityclass.scheduling.k8s.io",
	"clusterrolebindings.rbac.authorization.k8s.io",
	"clusterroles.rbac.authorization.k8s.io",
	"rolebindings.rbac.authorization.k8s.io",
	"roles.rbac.authorization.k8s.io",
	"daemonset.apps",
	"deployment.apps",
	"replicaset.apps",
	"statefulset.apps",
	"cronjob.batch",
	"job.batch",
	"persistentvolumeclaim",
	"persistentvolume",
	"pod",
	"service",
	"endpoints",
}

// ErrNotHandled is returned when a resource is not handled
var ErrNotHandled = fmt.Errorf("resource not handled")
